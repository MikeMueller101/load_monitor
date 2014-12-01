package main

import (
	"alertservice"
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"queue"
	"runtime"
	"strconv"
	"time"
)

const debug = true
const maxSizeCpuLoadQueue = 60 // sets the max size of the cpuload queue, stores the cpu load over the past 10 minutes

type Msg struct {
	MessageId string
	Content   string
	TimeStamp int64
}

// Client connection consists of the websocket and the client ip
type Client struct {
	websocket *websocket.Conn
	clientIP  string
}

type LoadMonitor struct {
	alertservice   *alertservice.AlertService
	errChan        chan error // unbuffered channel
	errChanWebsock chan error // unbuffered channel
	activeClients  map[string]Client
	measureChan1   chan float64 // buffered channel with 10 entries, this needs to be adpated depending on the number of clients
	measureChan2   chan float64 // buffered channel with 10 entries, this needs to be adpated depending on the number of clients
	alertChan      chan string  // buffered channel with 10 entries, this needs to be adpated depending on the number of clients
	newClientChan  chan Client  // buffered channel with 10 entries, this needs to be adpated depending on the number of clients
	alertQueue     *queue.Queue
	cpuLoadQueue   *queue.Queue
}

func NewLoadMonitor() *LoadMonitor {
	m := LoadMonitor{}
	m.alertservice = alertservice.New()
	m.activeClients = make(map[string]Client)
	m.errChan = make(chan error)
	m.measureChan1 = make(chan float64, 10)
	m.measureChan2 = make(chan float64, 10)
	m.alertChan = make(chan string, 10)
	m.newClientChan = make(chan Client, 10)
	m.alertQueue = queue.NewQueue()
	m.cpuLoadQueue = queue.NewQueue()
	return &m
}

func IntToString(value int) string {
	return strconv.FormatInt(int64(value), 10)
}

func StringToFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func StringToInt(value string) int {
	result, _ := strconv.ParseInt(value, 10, 64)
	return int(result)
}

func FloatToString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func getNumCpus() int {
	cmdStr := "sysctl hw.ncpu | awk '{print $2}'"
	out, _ := exec.Command("/bin/sh", "-c", cmdStr).Output()

	numCpus := string(out[:len(out)-1])
	return StringToInt(numCpus)
}

func normalizeLoadAvg(load float64, numCpus int, max float64) float64 {
	normLoad := load / float64(numCpus)
	if normLoad > max {
		normLoad = max
	}
	return Round(normLoad, 0.5, 3)
}

func getLoadAvg() string {
	cmdStr := "uptime | grep -ohe 'load average[s:][: ].*' | awk '{ print $3 }'"
	out, _ := exec.Command("/bin/sh", "-c", cmdStr).Output()

	return string(out[:len(out)-1])
}

func (m *LoadMonitor) measureCpuLoad() {
	var loadAvg float64
	numCpus := getNumCpus()

	for {
		val := getLoadAvg()
		loadAvg = normalizeLoadAvg(StringToFloat(val), numCpus, 2.0)

		if debug {
			fmt.Println(loadAvg)
		}

		m.measureChan1 <- loadAvg
		m.measureChan2 <- loadAvg

		time.Sleep(10 * time.Second)
	}
}

func (m *LoadMonitor) startAlertService() {
	for {
		val := <-m.measureChan2

		detected, msg := m.alertservice.DetectAlert(val)
		if detected {
			m.alertChan <- msg
		}
	}
}

func (m *LoadMonitor) sendClientMsg(msg *Msg, ip string) {
	var err error
	var Message = websocket.JSON

	if err = Message.Send(m.activeClients[ip].websocket, msg); err != nil {
		// we could not send the message to a peer
		log.Println("Could not send message to:", ip, err.Error())
		log.Println("Client disconnected:", ip)
		delete(m.activeClients, ip)
	}
}

func (m *LoadMonitor) sendBroadcastMsg(msg *Msg) {
	var err error
	var Message = websocket.JSON

	for ip, _ := range m.activeClients {
		if err = Message.Send(m.activeClients[ip].websocket, msg); err != nil {
			// we could not send the message to a peer
			log.Println("Could not send message to:", ip, err.Error())
			log.Println("Client disconnected:", ip)
			delete(m.activeClients, ip)
		}
	}
}

func (m *LoadMonitor) sendQueueData(ip string) {
	for i := 0; i < m.cpuLoadQueue.Len(); i++ {
		e, found := m.cpuLoadQueue.Get(i)

		if found {
			if msg, ok := e.(*Msg); ok {
				m.sendClientMsg(msg, ip)
			}
		}
	}

	for i := 0; i < m.alertQueue.Len(); i++ {
		e, found := m.alertQueue.Get(i)

		if found {
			if msg, ok := e.(*Msg); ok {
				m.sendClientMsg(msg, ip)
			}
		}
	}
}

// this routine handles all outgoing websocket messages
func (m *LoadMonitor) pushDataToClients() {
	for {
		select {
		// a new Client is connecting
		case newClient := <-m.newClientChan:
			// send current Queue data to the new connecting client
			m.activeClients[newClient.clientIP] = newClient
			m.sendQueueData(newClient.clientIP)

		// broadcast a new CPU load message to all clients
		case newCpuLoad := <-m.measureChan1:
			msg := Msg{"Plot", FloatToString(newCpuLoad), time.Now().UnixNano() / int64(time.Millisecond)}
			m.sendBroadcastMsg(&msg)
			// add msg to cpuLoadQueue
			if m.cpuLoadQueue.Len() < maxSizeCpuLoadQueue {
				m.cpuLoadQueue.Push(&msg)
			} else {
				m.cpuLoadQueue.Pop()
				m.cpuLoadQueue.Push(&msg)
			}

		// broadcast an alert message to all clients
		case newAlert := <-m.alertChan:
			msg := Msg{"Alert", newAlert, time.Now().UnixNano() / int64(time.Millisecond)}
			m.sendBroadcastMsg(&msg)
			// add msg to alertQueue
			m.alertQueue.Push(&msg)
		}
	}
}

// reference: https://github.com/Niessy/websocket-golang-chat
// WebSocket server to handle clients
func (m *LoadMonitor) WebSocketServer(ws *websocket.Conn) {
	var err error

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	if debug {
		log.Println("New client connected:", client)
	}

	m.newClientChan <- Client{ws, client}

	// wait for errChan, so the websocket stays open otherwise it'll close
	err = <-m.errChanWebsock
}

// handler for the main page
func HomeHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	webpage, err := ioutil.ReadFile("home.html")

	if err != nil {
		http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
	}

	fmt.Fprint(response, string(webpage))
}

func (m *LoadMonitor) startHTTPServer() {
	http.Handle("/", http.HandlerFunc(HomeHandler))
	http.Handle("/sock", websocket.Handler(m.WebSocketServer))

	err := http.ListenAndServe(":8080", nil)
	m.errChanWebsock <- err
	m.errChan <- err
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	m := NewLoadMonitor()

	go m.startHTTPServer()
	go m.pushDataToClients()
	go m.measureCpuLoad()
	go m.startAlertService()

	err := <-m.errChan
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
