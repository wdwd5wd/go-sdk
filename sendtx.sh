# shardnum=2
# for ((i=0; i<5; i ++))
# do
#     port=`expr 9500 + $i \* $shardnum`
#     ./hmy "--node=http://172.31.20.144:${port}" transfer --file "testtxs${i}.json" &
#     # sleep 1
# done

# wait

# ./hmy --node=http://172.31.20.144:9500 transfer --file testtxs.json &
# ./hmy --node=http://172.31.20.144:9502 transfer --file testtxs9502.json &

# wait