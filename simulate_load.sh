#!/bin/bash

start() { 
  # CPU
  # use 1 2 3 4 for quad core, etc.
  for cpu in 1 2 3 4; do
     yes > /dev/null &
     #( while true; do true; done ) &
  done
}


stop() {
  killall yes
}

case $1 in start|stop) "$1" ;; *) printf >&2 '%s: unknown command\n' "$1"; exit 1;; esac