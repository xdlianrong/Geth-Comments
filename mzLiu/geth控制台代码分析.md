###### 1.goja

goja 是一个 Go 实现的 ECMAScript 5.1(+)。

它不是 V8 或 SpiderMonkey 或任何其他通用 JavaScript 引擎的替代品，因为它更慢。它可以作为一种嵌入式脚本语言使用，或者可以作为避免非 Go 相关性的一种方式。

灵感来源于 [otto](https://github.com/robertkrimen/otto) 。完全支持 ECMAScript 5.1，通过几乎所有用 es5id 标记的 tc39 测试，平均比 otto 快6-7倍，同时使用相当少的内存。

简单的理解是，在go里面写javascript，这样就可以以脚本的方式实现控制台，并和geth实时交互。

###### 2.RPC





