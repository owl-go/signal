#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

BIZ=biz
ISLB=islb
SFU=sfu
ISSR=issr

echo "------------------stop $BIZ------------------"
echo "pkill $BIZ"
pkill $BIZ

echo "------------------stop $ISLB------------------"
echo "pkill $ISLB"
pkill $ISLB

echo "------------------stop $SFU------------------"
echo "pkill $SFU"
pkill $SFU

echo "------------------stop $ISSR------------------"
echo "pkill $ISSR"
pkill $ISSR
