#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

BIZ_BIN=biz
DIST_BIN=dist
ISLB_BIN=islb
SFU_BIN=sfu

BUILD_PATH1=$APP_DIR/bin/$BIZ_BIN
BUILD_PATH2=$APP_DIR/bin/$DIST_BIN
BUILD_PATH3=$APP_DIR/bin/$ISLB_BIN
BUILD_PATH4=$APP_DIR/bin/$SFU_BIN

help(){
    echo ""
    echo "build script"
    echo "Usage: ./build.sh [-h]"
    echo ""
}

while getopts "o:h" arg
do
    case $arg in
        h)
            help;
            exit 0
            ;;
        o)
            OS_TYPE=$OPTARG
            ;;
        ?)
            echo "No argument needed. Ignore them all!"
            ;;
    esac
done

if [[ "$OS_TYPE" == "Darwin" || "$OS_TYPE" == "darwin" || "$OS_TYPE" == "mac" ]];then
    echo "GO Target Arch: " $OS_TYPE
    export CGO_ENABLED=0
    export GOOS=darwin
fi

if [[ "$OS_TYPE" == "Linux" || "$OS_TYPE" == "linux" ]];then
    echo "GO Target Arch: " $OS_TYPE
    export CGO_ENABLED=0
    export GOOS=linux
fi


echo "------------------build $BIZ_BIN------------------"
echo "go build -o $BUILD_PATH1"
cd $APP_DIR/cmd/biz
go build -tags netgo -o $BUILD_PATH1


echo "------------------build $DIST_BIN------------------"
echo "go build -o $BUILD_PATH2"
cd $APP_DIR/cmd/dist
go build -tags netgo -o $BUILD_PATH2


echo "------------------build $ISLB_BIN------------------"
echo "go build -o $BUILD_PATH3"
cd $APP_DIR/cmd/islb
go build -tags netgo -o $BUILD_PATH3

echo "------------------build $SFU_BIN------------------"
echo "go build -o $BUILD_PATH4"
cd $APP_DIR/cmd/sfu
go build -tags netgo -o $BUILD_PATH4
