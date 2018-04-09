#!/bin/sh

# This script was used for evaluating the different configuration. It was added to the repository for archive purpose

SC_EXEC="/home/scion/go/src/github.com/Meldanor/SCIONLab_SpeedCam/core"
RESULT_DIR="/home/scion/results"
STD_ARGS="-psUrl=http://localhost:8082/pathServerRequests -brUrl=http://localhost:8082/prometheusClient"

echo "Prepare directories and configs"

# Configs

# Standard config - linear of 0.1 with waittime of 10 seconds between inspections
STANDARD_DIR="$RESULT_DIR/standard"
mkdir $STANDARD_DIR -p
STANDARD="-resultDir=$STANDARD_DIR"

# Always use 1 node
CONST_DIR="$RESULT_DIR/const"
mkdir $CONST_DIR -p
CONST="-resultDir=$CONST_DIR -scaleType=const -scaleParam=1"

# Linear with 0.1 as scale factor
LINEAR01_DIR="$RESULT_DIR/linear01"
mkdir $LINEAR01_DIR -p
LINEAR01="-resultDir=$LINEAR01_DIR -scaleType=linear -scaleParam=0.1"

# Linear with 0.2 as scale factor
LINEAR02_DIR="$RESULT_DIR/linear02"
mkdir $LINEAR02_DIR -p
LINEAR02="-resultDir=$LINEAR02_DIR -scaleType=linear -scaleParam=0.2"

# Linear with 0.33 as scale factor
LINEAR033_DIR="$RESULT_DIR/linear033"
mkdir $LINEAR033_DIR -p
LINEAR033="-resultDir=$LINEAR033_DIR -scaleType=linear -scaleParam=0.33"

# Linear with 0.33 as scale factor
LOG_DIR="$RESULT_DIR/log"
mkdir $LOG_DIR -p
LOG="-resultDir=$LOG_DIR -scaleType=log -scaleParam=10"

# Random wait time
RANDOM_DIR="$RESULT_DIR/random"
mkdir $RANDOM_DIR -p
RANDOM="-resultDir=$RANDOM_DIR -intervalStrat=random"

# Fixed wait time of 60 seconds
FIXED_DIR="$RESULT_DIR/fixed"
mkdir $FIXED_DIR -p
FIXED="-resultDir=$FIXED_DIR -intervalStrat=fixed -intervalMin=60"

# Experience
EXPERIENCE_DIR="$RESULT_DIR/experience"
mkdir $EXPERIENCE_DIR -p
EXPERIENCE="-resultDir=$EXPERIENCE_DIR -intervalStrat=experience"

# Experience with 12 episodes (more)
EXPERIENCE12_DIR="$RESULT_DIR/experience_12"
mkdir $EXPERIENCE12_DIR -p
EXPERIENCE12="-resultDir=$EXPERIENCE12_DIR -intervalStrat=experience -cEpisodes=12"

# Experience with 3 episodes (fewer)
EXPERIENCE03_DIR="$RESULT_DIR/experience_03"
mkdir $EXPERIENCE03_DIR -p
EXPERIENCE03="-resultDir=$EXPERIENCE03_DIR -intervalStrat=experience -cEpisodes=3"


echo "Start different configuration"

echo "Start STANDARD"
nohup $SC_EXEC $STD_ARGS $STANDARD > $STANDARD_DIR/output.log 2>&1 error.log &
echo $! > $STANDARD_DIR/pid

echo "Start CONST"
nohup $SC_EXEC $STD_ARGS $CONST > $CONST_DIR/output.log 2>&1 error.log &
echo $! > $CONST_DIR/pid

echo "Start LINEAR01"
nohup $SC_EXEC $STD_ARGS $LINEAR01 > $LINEAR01_DIR/output.log 2>&1 error.log &
echo $! > $LINEAR01_DIR/pid

echo "Start LINEAR02"
nohup $SC_EXEC $STD_ARGS $LINEAR02 > $LINEAR02_DIR/output.log 2>&1 error.log &
echo $! > $LINEAR02_DIR/pid

echo "Start LINEAR033"
nohup $SC_EXEC $STD_ARGS $LINEAR033 > $LINEAR033_DIR/output.log 2>&1 error.log &
echo $! > $LINEAR033_DIR/pid

echo "Start LOG"
nohup $SC_EXEC $STD_ARGS $LOG > $LOG_DIR/output.log 2>&1 error.log &
echo $! > $LOG_DIR/pid

echo "Start RANDOM"
nohup $SC_EXEC $STD_ARGS $RANDOM > $RANDOM_DIR/output.log 2>&1 error.log &
echo $! > $RANDOM_DIR/pid

echo "Start FIXED"
nohup $SC_EXEC $STD_ARGS $FIXED > $FIXED_DIR/output.log 2>&1 error.log &
echo $! > $FIXED_DIR/pid

echo "Start Standard EXPERIENCE"
nohup $SC_EXEC $STD_ARGS $EXPERIENCE > $EXPERIENCE_DIR/output.log 2>&1 error.log &
echo $! > $EXPERIENCE_DIR/pid

echo "Start EXPERIENCE 12 episodes"
nohup $SC_EXEC $STD_ARGS $EXPERIENCE12 > $EXPERIENCE12_DIR/output.log 2>&1 error.log &
echo $! > $EXPERIENCE12_DIR/pid

echo "Start Experience 03 episodes"
nohup $SC_EXEC $STD_ARGS $EXPERIENCE03 > $EXPERIENCE03_DIR/output.log 2>&1 error.log &
echo $! > $EXPERIENCE03_DIR/pid


echo "All started"

echo "Start monitoring via top()"
top -d 5 -p $(cat $RESULT_DIR/**/pid | tr '\n' ',' | sed 's/.$//') -b > $RESULT_DIR/benchmark.txt &


