
#!/usr/bin/bash

root_proj=`pwd`
export GOPATH=$GOPATH:$root_proj

go install
