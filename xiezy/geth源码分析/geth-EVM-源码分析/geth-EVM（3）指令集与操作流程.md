# EVM源码分析（指令集与操作流程）

## 1 概述

1. 编写合约

2. 再生成汇编代码/十六进制字节码

   汇编代码可以通过汇编的指令集opcode生成十六进制字节码

   Binary为最后编译出来的十六进制字节码(部署代码+runtime代码+auxdata)

   部署代码：创建合约交易中运行的代码

   runtime代码：外部账户调用合约过程中运行的代码

   auxdata是合约代码的校验码和solc版本的数据，并且运行合约时是没使用到。

3. evm创建合约时，运行Binary

   实际运行部署代码，runtime和auxdata字段最终存储到对应账户的stateDB中

   后序可以进一步查看合约成功后生成的智能合约ABI

4. evm调用合约时，运行runtime字段

   

## 2 执行流程

这里从运行中的run方法入手，下面是run的主要片段（core/vm/interpreter.go）

```go
// 开始循环PC计数执行 直到有中止执行或者跳出循环
   for atomic.LoadInt32(&in.evm.abort) == 0 {
      if in.cfg.Debug {
         // Capture pre-execution values for tracing.
         logged, pcCopy, gasCopy = false, pc, contract.Gas
      }
      // Get the operation from the jump table and validate the stack to ensure there are
      // enough stack items available to perform the operation.
      // evm通过相应指令在core/vm/opcodes.go文件中找到对应的操作码
      op = contract.GetOp(pc)
      // 根据操作码获取对应以太坊链阶段的操作函数
      operation := in.cfg.JumpTable[op]
      if !operation.valid {
         return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
      }
      // 验证栈中的数据是否符合操作码需要的数据
      if sLen := stack.len(); sLen < operation.minStack {
         return nil, fmt.Errorf("stack underflow (%d <=> %d)", sLen, operation.minStack)
      } else if sLen > operation.maxStack {
         return nil, fmt.Errorf("stack limit reached %d (%d)", sLen, operation.maxStack)
      }
      // If the operation is valid, enforce and write restrictions
      if in.readOnly && in.evm.chainRules.IsByzantium {
         // If the interpreter is operating in readonly mode, make sure no
         // state-modifying operation is performed. The 3rd stack item
         // for a call operation is the value. Transferring value from one
         // account to the others means the state is modified and should also
         // return with an error.
         if operation.writes || (op == CALL && stack.Back(2).Sign() != 0) {
            return nil, errWriteProtection
         }
      }
      // 计算对应指令所要消耗的gas，这里是静态气体，是固定的
      cost = operation.constantGas // For tracing
      if !contract.UseGas(operation.constantGas) {
         return nil, ErrOutOfGas
      }

      var memorySize uint64
      // 有些指令是需要额外的内存消耗 在jump_table.go文件中可以看到他们具体每个操作码的对应的额外内存消耗计算
		// 并不是所有的指令都需要计算消耗的内存 
		// memorySize指向对应的计算消耗内存的函数 根据消耗的内存来计算消费的gas
      if operation.memorySize != nil {
         memSize, overflow := operation.memorySize(stack)
         if overflow {
            return nil, errGasUintOverflow
         }
         // memory is expanded in words of 32 bytes. Gas
         // is also calculated in words.
         if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
            return nil, errGasUintOverflow
         }
      }
      // 计算此内存占用花费的gas数量，这里是动态气体，是可变的
      if operation.dynamicGas != nil {
         var dynamicCost uint64
         dynamicCost, err = operation.dynamicGas(in.evm, contract, stack, mem, memorySize)
         cost += dynamicCost // total cost, for debug tracing
         if err != nil || !contract.UseGas(dynamicCost) {
            return nil, ErrOutOfGas
         }
      }
      if memorySize > 0 {
         mem.Resize(memorySize)
      }

      if in.cfg.Debug {
         in.cfg.Tracer.CaptureState(in.evm, pc, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
         logged = true
      }

      // 开始执行此操作码对应的操作函数，同时会返回执行结果同时也会更新PC计数器 
	  // 大部分的操作码对应的操作函数都是在instructions.go中可以找得到
      res, err = operation.execute(&pc, in, contract, mem, stack)
      // 释放intpool
      if verifyPool {
         verifyIntegerPool(in.intPool)
      }
      // 如果这个操作码是一个返回参数 那么就把需要的内容写入returnData
      if operation.returns {
         in.returnData = res
      }

      // 到这里也就意味着一个操作码已经执行完成了，应该根据这次的执行结果来决定下一步的动作
      // 1. 如果执行出错了，直接返回错误
      // 2. 如果只能合约代码中止了(比如断言失败)那么直接返回执行结果 
      // 3. 如果是暂停指令，则直接返回结果
      // 4. 如果操作符不是一个跳转，则直接PC指向下一个指令，继续循环执行
      switch {
      case err != nil:
         return nil, err
      case operation.reverts:
         return res, errExecutionReverted
      case operation.halts:
         return res, nil
      case !operation.jumps:
         pc++
      }
   }
```

## 3 操作码获取指令

core/vm/opcodes.go：部分代码，实际上就是一堆的常量操作符（操作码）

这些操作符会在core/vm/jump_table.go这个文件中映射成对应的操作函数，用于执行

```go
func (c *Contract) GetOp(n uint64) OpCode {
   return OpCode(c.GetByte(n))
}

// GetByte返回合约字节数组中的第n个字节
func (c *Contract) GetByte(n uint64) byte {
   if n < uint64(len(c.Code)) {
      return c.Code[n]
   }

   return 0
}
```

下面举一段core/vm/opcodes.go中的操作码实例

```go
// 0x0 range - arithmetic ops.
const (
	STOP OpCode = iota      // iota定义从十六进制0开始的自增型常量
	ADD
	MUL
	SUB
	DIV
	SDIV
	MOD
	SMOD
	ADDMOD
	MULMOD
	EXP
	SIGNEXTEND
)
// 0x10 range - comparison ops.
const (
   LT OpCode = iota + 0x10	// iota定义从16进制10开始的自增型常量
   GT						// 0x11
   SLT						// 0x12
   SGT						// 0x13 
   EQ						// 0x14
   ISZERO					// 0x15
   AND
   OR
   XOR
   NOT
   BYTE          			// 0x1a
   SHL						// 0x1b
   SHR						// 0x1c
   SAR						// 0x1d

   SHA3 = 0x20
)
```

## 4 指令的操作对象

### 4.1 数据结构

operation的数据结构 （core/vm/jump_table.go）

下列中的定义的函数在同文件（core/vm/jump_table.go）中定义了模板 

```go
type operation struct {
   execute     executionFunc   // 指令对应的执行函数（important）
   constantGas uint64          // 静态gas
   dynamicGas  gasFunc         // 动态gas
   minStack int                // pops（其实就是指令用到的stack中的item重量） 
   maxStack int				   // 1024 + pop - push（规定变量，保证堆栈在操作过程中不溢出）
   memorySize memorySizeFunc   // 返回操作所需的内存大小

   halts   bool // 指示操作是否应停止进一步执行
   jumps   bool // 指示程序计数器是否不应递增
   writes  bool // 确定此操作是否为状态修改操作
   valid   bool // 指示检索的操作是否有效和已知
   reverts bool // 确定操作是否恢复状态(隐式停止)
   returns bool // 确定操作是否设置返回的数据内容
}
```

下面是对operation中的Func定义的模板。在一个具体的指令中，下面内容都将被实例化

```go
type (
   executionFunc func(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error)
   gasFunc       func(*EVM, *Contract, *Stack, *Memory, uint64) (uint64, error)
   memorySizeFunc func(*Stack) (size uint64, overflow bool)
)
```

### 4.2 根据evm版本生成具体操作对象

下面，根据当前的版本选择对应的指令生成函数

代码中主要函数返回值的定义都在FrontierInstructionSet阶段写明，供其他阶段调用

```go
var (
	frontierInstructionSet       = newFrontierInstructionSet()
	homesteadInstructionSet      = newHomesteadInstructionSet()
	byzantiumInstructionSet      = newByzantiumInstructionSet()
	constantinopleInstructionSet = newConstantinopleInstructionSet()
)
```

下面是具体frontierInstructionSet阶段对应生成的操作对象

```go
func newFrontierInstructionSet() JumpTable {
   return JumpTable{
      STOP: {
         execute:     opStop,
         constantGas: 0,
         minStack:    minStack(0, 0),
         maxStack:    maxStack(0, 0),
         halts:       true,
         valid:       true,
      },
      ADD: {
         execute:     opAdd,
         constantGas: GasFastestStep,
         minStack:    minStack(2, 1),
         maxStack:    maxStack(2, 1),
         valid:       true,
      },
      MUL: {
         execute:     opMul,
         constantGas: GasFastStep,
         minStack:    minStack(2, 1),
         maxStack:    maxStack(2, 1),
         valid:       true,
      },
      SUB: {
         execute:     opSub,
         constantGas: GasFastestStep,
         minStack:    minStack(2, 1),
         maxStack:    maxStack(2, 1),
         valid:       true,
      },
      ......
       instructionSet[REVERT] = operation{
		execute:    opRevert,
		dynamicGas: gasRevert,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
		memorySize: memoryRevert,
		valid:      true,
		reverts:    true,
		returns:    true,
	  },
      ......
      
   }
}
```

附：Jumptable的数据类型是一个长度256的operation数组，evm指令最多为2的8次方个

`JumpTable [256]operation`

## 5 操作对象中的具体执行函数

具体执行函数在操作对象中被定义为execute字段

操作对象中的执行函数在core/vm/instructions.go中被定义，execute中的两个实例如下

```go
// 先看一个只涉及stack的执行函数，实现两个数相加
func opAdd(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
   x, y := stack.pop(), stack.peek()   // 注意这里的y最后不删除
   math.U256(y.Add(x, y))        // 无符号256位整形运算，y最后赋值为x+y
   interpreter.intPool.put(x)    // 在intpool对象中销毁big.int型整数x
   return nil, nil
}
// 再看一个涉及到memory的执行函数，可能是函数的跳转复原，实现地址+偏移地址操作
func opRevert(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	interpreter.intPool.put(offset, size)
    // 一般涉及memory的执行函数都是要有返回值的
	return ret, nil
}
// 下面是把合约中第一个变量对应的操作数加入栈中的操作，
func opPush1(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		codeLen = uint64(len(contract.Code))
		integer = interpreter.intPool.get()
	)
    // 和前两个不一样是这里调用了pc指针，取得了Push1操作码的后一字节，也就是操作数
	*pc += 1
	if *pc < codeLen {
		stack.push(integer.SetUint64(uint64(contract.Code[*pc])))
	} else {
		stack.push(integer.SetUint64(0))
	}
	return nil, nil
}
```

附：memory中的GetPtr方法

```go
// GetPtr returns the offset + size
func (m *Memory) GetPtr(offset, size int64) []byte {
   if size == 0 {
      return nil
   }

   if len(m.store) > int(offset) {
      return m.store[offset : offset+size]     // store就是一个定义在memory中的[]byte
   }

   return nil
}
```

## 6 gas消耗

### 6.1 静态gas消耗

操作对象operation的constantGas字段定义了指令默认消耗的gas费用

在项目静态参数配置中的params/protocol_params.go中同样定义了一些gas消耗

当然更多指令还是调用了core/vm/gas.go下面的参数，如下所示

```go
// Gas costs
const (
   GasQuickStep   uint64 = 2
   GasFastestStep uint64 = 3
   GasFastStep    uint64 = 5
   GasMidStep     uint64 = 8
   GasSlowStep    uint64 = 10
   GasExtStep     uint64 = 20
)
```

### 6.2 动态gas消耗

操作对象operation的dynamicGas字段定义了指令调用内存消耗的gas费用

在core/vm/gas_table.go中有详细的定义，动态的gas消耗还是基于调用过程中占用内存的大小（即operation中的memorySize字段），如下图所示。

当然也有一些指令定义的dynamicGas会在memoryGasCost基础上进行进一步修改

```go
func memoryGasCost(mem *Memory, newMemSize uint64) (uint64, error) {
   if newMemSize == 0 {
      return 0, nil
   }
   // The maximum that will fit in a uint64 is max_word_count - 1. Anything above
   // that will result in an overflow. Additionally, a newMemSize which results in
   // a newMemSizeWords larger than 0xFFFFFFFF will cause the square operation to
   // overflow. The constant 0x1FFFFFFFE0 is the highest number that can be used
   // without overflowing the gas calculation.
   if newMemSize > 0x1FFFFFFFE0 {
      return 0, errGasUintOverflow
   }
   newMemSizeWords := toWordSize(newMemSize)
   newMemSize = newMemSizeWords * 32

   if newMemSize > uint64(mem.Len()) {
      square := newMemSizeWords * newMemSizeWords
      linCoef := newMemSizeWords * params.MemoryGas
      quadCoef := square / params.QuadCoeffDiv
      newTotalFee := linCoef + quadCoef

      // 重要是这里，最新占用总内存消耗的gas - 前面过程中占用内存消耗的gas
      fee := newTotalFee - mem.lastGasCost
      // 更新memory中的lastGasCost字段
      mem.lastGasCost = newTotalFee

      return fee, nil
   }
   return 0, nil
}
```

















