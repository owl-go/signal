#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

BIZ=biz
DIST=dist
ISLB=islb
SFU=sfu

BUILD_PATH1=$APP_DIR/bin/$BIZ
BUILD_PATH2=$APP_DIR/bin/$DIST
BUILD_PATH3=$APP_DIR/bin/$ISLB
BUILD_PATH4=$APP_DIR/bin/$SFU

BIZ_LOG=$APP_DIR/logs/$BIZ.log
DIST_LOG=$APP_DIR/logs/$DIST.log
ISLB_LOG=$APP_DIR/logs/$ISLB.log
SFU_LOG=$APP_DIR/logs/$SFU.log

echo "------------------delete $BIZ------------------"
echo "rm $BUILD_PATH1"
rm $BUILD_PATH1

echo "------------------delete $DIST------------------"
echo "rm $BUILD_PATH2"
rm $BUILD_PATH2

echo "------------------delete $ISLB------------------"
echo "rm $BUILD_PATH3"
rm $BUILD_PATH3

echo "------------------delete $SFU------------------"
echo "rm $BUILD_PATH4"
rm $BUILD_PATH4

echo "------------------delete $BIZ LOG------------------"
echo "rm $BIZ_LOG"
rm $BIZ_LOG

echo "------------------delete $DIST LOG------------------"
echo "rm $DIST_LOG"
rm $DIST_LOG

echo "------------------delete $ISLB LOG------------------"
echo "rm $ISLB_LOG"
rm $ISLB_LOG

echo "------------------delete $SFU LOG------------------"
echo "rm $SFU_LOG"
rm $SFU_LOG
