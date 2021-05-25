#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

BIZ=biz
ISLB=islb
SFU=sfu
ISSR=issr

BUILD_PATH1=$APP_DIR/bin/$BIZ
BUILD_PATH3=$APP_DIR/bin/$ISLB
BUILD_PATH4=$APP_DIR/bin/$SFU
BUILD_PATH5=$APP_DIR/bin/$ISSR

BIZ_LOG=$APP_DIR/logs/$BIZ.log
ISLB_LOG=$APP_DIR/logs/$ISLB.log
SFU_LOG=$APP_DIR/logs/$SFU.log
ISSR_LOG=$APP_DIR/logs/$ISSR.log

echo "------------------delete $BIZ------------------"
echo "rm $BUILD_PATH1"
rm $BUILD_PATH1

echo "------------------delete $ISLB------------------"
echo "rm $BUILD_PATH3"
rm $BUILD_PATH3

echo "------------------delete $SFU------------------"
echo "rm $BUILD_PATH4"
rm $BUILD_PATH4

echo "------------------delete $ISSR------------------"
echo "rm $BUILD_PATH5"
rm $BUILD_PATH5

echo "------------------delete $BIZ LOG------------------"
echo "rm $BIZ_LOG"
rm $BIZ_LOG

echo "------------------delete $ISLB LOG------------------"
echo "rm $ISLB_LOG"
rm $ISLB_LOG

echo "------------------delete $SFU LOG------------------"
echo "rm $SFU_LOG"
rm $SFU_LOG

echo "------------------delete $ISSR LOG------------------"
echo "rm $ISSR_LOG"
rm $ISSR_LOG
