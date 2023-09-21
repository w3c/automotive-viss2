---
title: "VISSv2 Build System"
---

## Installing Golang

Most of the code at this repository is written in Golang, so in order to use this repo Golang must be installed on the computer.

Searching for "install golang" gives many hits, of which this is one:

[How to install Go (golang) on Ubuntu Linux](https://www.cyberciti.biz/faq/how-to-install-gol-ang-on-ubuntu-linux/).

For other operating systems [this](https://go.dev/doc/install) may be helpful.

## Building and running

As several of the Golang based Sw components on this repo can be started with command line input to configure its behavior,
it is suitable to first build it (in the directory of the source code)

$ go build

and then run it

$ ./'name-of-executable' 'any-command-line-config-input'

If the SwC supports command line configuration input it is possible to add "-h" (no quotes) on the command line, which will then show a help text.
Checking the first lines of the main() function in the source code is another possibility to find out.
If there is any calls to the "github.com/akamensky/argparse" lib then it is used.

As the configurations have a default it is always possible to run it without adding any comand line configuration input.
The configuration possibilities of the different SwCs are described in the respective chapters of this tutorial.
