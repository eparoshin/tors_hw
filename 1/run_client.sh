#!/usr/bin/env bash

sudo docker run -i --network tors_1_network -P client_container "$@"
