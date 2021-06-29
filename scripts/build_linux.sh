#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

export GOPROXY="https://goproxy.cn,direct"

BIZ_BIN=biz
ISLB_BIN=islb
SFU_BIN=sfu
ISSR_BIN=issr

PROJECT=$1
OS_TYPE="linux"
BUILD_PATH1=$APP_DIR/bin/$BIZ_BIN
BUILD_PATH3=$APP_DIR/bin/$ISLB_BIN
BUILD_PATH4=$APP_DIR/bin/$SFU_BIN
BUILD_PATH5=$APP_DIR/bin/$ISSR_BIN

help(){
    echo ""
    echo "build script"
    echo "Usage: ./build.sh biz|islb|sfu|issr|all"
    echo "Usage: ./build.sh [-h]"
    echo ""
}

build_biz()
{
    echo "------------------build $BIZ_BIN------------------"
    echo "go build -o $BUILD_PATH1"
    cd $APP_DIR/cmd/biz
    go build -tags netgo -o $BUILD_PATH1
}

build_islb()
{
    echo "------------------build $ISLB_BIN------------------"
    echo "go build -o $BUILD_PATH3"
    cd $APP_DIR/cmd/islb
    go build -tags netgo -o $BUILD_PATH3
}

build_sfu()
{
    echo "------------------build $SFU_BIN------------------"
    echo "go build -o $BUILD_PATH4"
    cd $APP_DIR/cmd/sfu
    go build -tags netgo -o $BUILD_PATH4
}

build_issr()
{
    echo "------------------build $ISSR_BIN------------------"
    echo "go build -o $BUILD_PATH5"
    cd $APP_DIR/cmd/issr
    go build -tags netgo -o $BUILD_PATH5
}

if [ $# -ne 1 ]
then
    help
    exit 1
fi

if [ "$OS_TYPE" == "Darwin" ] || [ "$OS_TYPE" == "darwin" ] || [ "$OS_TYPE" == "mac" ];then
    echo "GO Target Arch: " $OS_TYPE
    export CGO_ENABLED=0
    export GOOS=darwin
fi

if [ "$OS_TYPE" == "Linux" ] || [ "$OS_TYPE" == "linux" ];then
    echo "GO Target Arch: " $OS_TYPE
    export CGO_ENABLED=0
    export GOARCH=amd64
    export GOOS=linux
fi

case $PROJECT in
$BIZ_BIN)
    build_biz
    ;;
$ISLB_BIN)
    build_islb
    ;;
$SFU_BIN)
    build_sfu
    ;;
$ISSR_BIN)
    build_issr
    ;;
all)
    build_biz
    build_islb
    build_sfu
    build_issr
    ;;
*)
    help
    ;;
esac
