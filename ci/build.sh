#! /bin/bash

# Download dep and retrieve dependencies.

curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
dep ensure
