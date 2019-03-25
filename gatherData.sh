#!/bin/bash

#time in seconds
test_time=3600

start_time=`date +%s`
curr_time=`date +%s`
echo "ID,NAME,CPU,MEM"

while [ $(($curr_time-$start_time)) -lt $test_time ]
do
    docker stats --no-stream | tail -n +2 | tr -s ' ' | cut -d ' ' -f 1,2,3,4 | tr ' ' ',' | \
    while read i
    do
        echo $curr_time","$i
    done
    curr_time=`date +%s`

done
