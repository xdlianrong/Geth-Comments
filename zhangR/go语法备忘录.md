# go语法备忘录

## waitgroup

```go
type WaitGroup struct {
    // contains filtered or unexported fields
}

//设置需要等待的 Go 程数量
func (wg *WaitGroup) Add(delta int)

//Go 程计数器减 1
func (wg *WaitGroup) Done()

//阻塞等待所有 Go 程结束（等待 Go 程计数器变为 0）
func (wg *WaitGroup) Wait()

```

实例代码

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

func foo1() {
	defer wg.Done()
	fmt.Println("entry foo1")
	time.Sleep(2 * time.Second)
	fmt.Println("exit foo1")
}
func foo2() {
	defer wg.Done()
	fmt.Println("entry foo2")
	time.Sleep(4 * time.Second)
	fmt.Println("exit foo2")
}
func foo3() {
	defer wg.Done()
	fmt.Println("entry foo3")
	time.Sleep(8 * time.Second)
	fmt.Println("exit foo3")
}
func main() {
	fmt.Println("entry main")
	wg.Add(3)
	go foo1()
	go foo2()
	go foo3()
	fmt.Println("wg.Wait()")
	wg.Wait()
	fmt.Println("exit main")
}

输出：
entry main
wg.Wait()
// 进入顺序不确定，多次运行结果不一样
entry foo1
entry foo3
entry foo2
// 在每个foo中添加不同延时后可以实现1->2->3，如果中间延时相同，返回顺序也不一定与进入顺序相同
exit foo1
exit foo2
exit foo3
exit main
```

## atomic包的使用

 原子操作。顾名思义这类操作满足原子性，其执行过程不能被中断，这也就保证了同一时刻一个线程的执行不会被其他线程中断，也保证了多线程下数据操作的一致性。 

在atomic包中对几种基础类型提供了原子操作，包括int32，int64，uint32，uint64，uintptr，unsafe.Pointer。对于每一种类型，提供了五类原子操作分别是
* Add, 增加和减少
* CompareAndSwap, 比较并交换
* Swap, 交换
* Load , 读取
* Store, 存储

