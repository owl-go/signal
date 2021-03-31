#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

mkdir -p $APP_DIR/logs

BIZ=biz
DIST=dist
ISLB=islb
SFU=sfu
ISSR=issr

BIZ_CFG=$APP_DIR/configs/biz.toml
DIST_CFG=$APP_DIR/configs/dist.toml
ISLB_CFG=$APP_DIR/configs/islb.toml
SFU_CFG=$APP_DIR/configs/sfu.toml
ISSR_CFG=$APP_DIR/configs/issr.toml

BIZ_LOG=$APP_DIR/logs/$BIZ.log
DIST_LOG=$APP_DIR/logs/$DIST.log
ISLB_LOG=$APP_DIR/logs/$ISLB.log
SFU_LOG=$APP_DIR/logs/$SFU.log
ISSR_LOG=$APP_DIR/logs/$ISSR.log

BUILD_PATH1=$APP_DIR/bin/$BIZ
BUILD_PATH2=$APP_DIR/bin/$DIST
BUILD_PATH3=$APP_DIR/bin/$ISLB
BUILD_PATH4=$APP_DIR/bin/$SFU
BUILD_PATH5=$APP_DIR/bin/$ISSR

echo "------------------start $BIZ------------------"
echo "nohup $BUILD_PATH1 -c $BIZ_CFG >>$BIZ_LOG 2>&1 &"
nohup $BUILD_PATH1 -c $BIZ_CFG >>$BIZ_LOG 2>&1 &

echo "------------------start $DIST------------------"
echo "nohup $BUILD_PATH2 -c $DIST_CFG >>$DIST_LOG 2>&1 &"
nohup $BUILD_PATH2 -c $DIST_CFG >>$DIST_LOG 2>&1 &

echo "------------------start $ISLB------------------"
echo "nohup $BUILD_PATH3 -c $ISLB_CFG >>$ISLB_LOG 2>&1 &"
nohup $BUILD_PATH3 -c $ISLB_CFG >>$ISLB_LOG 2>&1 &

echo "------------------start $SFU------------------"
echo "nohup $BUILD_PATH4 -c $SFU_CFG >>$SFU_LOG 2>&1 &"
nohup $BUILD_PATH4 -c $SFU_CFG >>$SFU_LOG 2>&1 &


echo "------------------start $ISSR------------------"
echo "nohup $BUILD_PATH5 -c $ISSR_CFG >>$ISSR_LOG 2>&1 &"
nohup $BUILD_PATH5 -c $ISSR_CFG >>$ISSR_LOG 2>&1 &
