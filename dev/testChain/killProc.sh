#强制关闭测试所启动的进程
pid=$(ps x | grep geth | grep -v grep | awk '{print $1}')
for i in $pid
do
  kill -9 $i
done
screen -S exchange -X quit