#强制关闭测试所启动的进程
kill -9 $(lsof -i:8545 | awk '{print $2}')
kill -9 $(lsof -i:8546 | awk '{print $2}')
kill -9 $(lsof -i:8547 | awk '{print $2}')
kill -9 $(lsof -i:8548 | awk '{print $2}')
kill -9 $(lsof -i:8549 | awk '{print $2}')
kill -9 $(lsof -i:1323 | awk '{print $2}')
screen -S exchange -X quit