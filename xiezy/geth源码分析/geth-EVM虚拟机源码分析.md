# EVM虚拟机源码分析

## 1 简介

evm虚拟机用于处理和执行一笔交易，在代码中会有两种情况

1. 如果交易转入方的地址为null，则调用creat（）创建智能合约
2. 如果交易转入方的地址不为null，则调用creat（）创建智能合约

##  2 操作流程

具体的执行流程如下：

利用栈进行操作

memory：存储instructions在执行中的临时变量

storage：存储账户中的重要数据

gas avail：用来记录剩余汽油费



![](images\EVM_01.jpg)

## 3 代码实现

以上内容的代码实现在core/vm包中实现，整个vm包调用的入口在core/state_transaction.go中。 

### 3.1 调用入口core/state_transaction.go

ApplyTransaction函数先将交易信息录入数据库，再创建evm虚拟机，然后执行相关功能

```go
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
    // 调用 Transaction.AsMessage 将一个 Transaction 对象转换成 Message 对象
    // 可以理解为把交易对象变成evm虚拟机可以识别的相关对象
    msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
    if err != nil {
        return nil, 0, err
    }
    // context这个对象中主要包含了一些访问当前区块链数据的方法，传递给上下文
    context := NewEVMContext(msg, header, bc, author)
    // 创建以太坊虚拟机，里面包含了evm的相关机制和事务
    vmenv := vm.NewEVM(context, statedb, config, cfg)
    // 把创建好的evm虚拟机的事务应用于和当前交易状态相关
    _, gas, failed, err := ApplyMessage(vmenv, msg, gp)
    if err != nil {
        return nil, 0, err
    }

    ......
}
```

下面是ApplyMessage函数，直接调用下面的函数TransitionDb，把创建好的evm虚拟机的事务应用于和当前交易状态相关

```go
func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, uint64, bool, error) {
   return NewStateTransition(evm, msg, gp).TransitionDb()
}
```

下面TransitionDb函数主要实现的是

```go

func (st *StateTransition/**一笔交易中的状态信息**/) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
    // 检查交易的 Nonce 值是否正确
   if err = st.preCheck(); err != nil {
      return
   }
   msg := st.msg
   sender := vm.AccountRef(msg.From())
   homestead := st.evm.ChainConfig().IsHomestead(st.evm.BlockNumber)
   istanbul := st.evm.ChainConfig().IsIstanbul(st.evm.BlockNumber)
   contractCreation := msg.To() == nil

   // 从交易发送者账户中扣取规定量gas
   gas, err := IntrinsicGas(st.data, contractCreation, homestead, istanbul)
   // 如果发生报错，直接导致交易失败
   if err != nil {
      return nil, 0, false, err
   }
   if err = st.useGas(gas); err != nil {
      return nil, 0, false, err
   }

   var (
      evm = st.evm
      // vm errors do not effect consensus and are therefor
      // not assigned to err, except for insufficient balance
      // error.
      vmerr error
   )
   
   if contractCreation {
       // 当 contractCreation = nil 时默认为调用creat（）方法，创建合约操作
       // Creat方法将调用SetNonce方法完成nonce+1
      ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
   } else {
      // 当 contractCreation = "某个确定的合约账户地址时"
      // 先给当前交易者账户的 nonce + 1
      st.state.SetNonce(msg.From(), st.state.GetNonce(sender.Address())+1)
      // 再调用call（）方法，执行合约账户中的智能合约
      ret, st.gas, vmerr = evm.Call(sender, st.to(), st.data, st.gas, st.value)
   }
    // 创建交易和执行交易中的报错信息
   if vmerr != nil {
      log.Debug("VM returned with error", "err", vmerr)
      // The only possible consensus-error would be if there wasn't
      // sufficient balance to make the transfer happen. The first
      // balance transfer may never fail.
      // 这里直接产生共识机制的错误。如果账户没有足够余额，将不会退回gas，直接return
      if vmerr == vm.ErrInsufficientBalance {
         return nil, 0, false, vmerr
      }
   }
   // 退还剩余gas费，将交易方账户余额增加剩余的gas费
   st.refundGas()
   st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))

   return ret, st.gasUsed(), vmerr != nil, err
}
```

### 3.2 core/vm包

#### 3.2.1 core/vm包的目录如下

```
.
├── analysis.go            // 跳转目标判定
├── common.go
├── contract.go            // 合约的数据结构
├── contracts.go           // 预编译好的合约
├── errors.go
├── evm.go                 // 对外提供的接口   
├── gas.go                 // 用来计算指令耗费的 gas
├── gas_table.go           // 指令耗费计算函数表
├── gen_structlog.go       
├── instructions.go        // 指令操作
├── interface.go           // 定义 StateDB 的接口
├── interpreter.go         // 解释器
├── intpool.go             // 存放大整数
├── int_pool_verifier_empty.go
├── int_pool_verifier.go
├── jump_table.go           // 指令和指令操作（操作，花费，验证）对应表
├── logger.go               // 状态日志
├── memory.go               // EVM 内存
├── memory_table.go         // EVM 内存操作表，用来衡量操作所需内存大小
├── noop.go
├── opcodes.go              // 指令以及一些对应关系     
├── runtime
│   ├── env.go              // 执行环境 
│   ├── fuzz.go
│   └── runtime.go          // 运行接口，测试使用
├── stack.go                // 栈
└── stack_table.go          // 栈验证
```

#### 3.2.2 数据结构

在 EVM 模块中，有两个高层次的结构体，分别是 Context，EVM。

context是函数之间保存的上下文参数信息

```go
type Context struct {
   // CanTransfer returns whether the account contains
   // sufficient ether to transfer the value
   CanTransfer CanTransferFunc
   // Transfer transfers ether from one account to the other
   Transfer TransferFunc
   // 返回第 n 个区块的哈希值
   GetHash GetHashFunc 

   // Message information
   Origin   common.Address // Provides information for ORIGIN
   GasPrice *big.Int       // Provides information for GASPRICE

   // Block information
   Coinbase    common.Address // Provides information for COINBASE
   GasLimit    uint64         // Provides information for GASLIMIT
   BlockNumber *big.Int       // Provides information for NUMBER
   Time        *big.Int       // Provides information for TIME
   Difficulty  *big.Int       // Provides information for DIFFICULTY
}
```

evm是以太坊虚拟机的基础对象，提供运行时的必要工具

一旦执行出错扣除所有汽油费，任何错误都将影响整体的代码，编写代码要谨慎

```go
type EVM struct {
   //就是上面的数据结构
   Context
   // StateDB是状态存储接口。这个接口非常重要。可以肯定的说一直evm中的大部分工作都是围绕这次接口进行的。
   StateDB StateDB 
   // 当前调用在栈中的深度
   depth int 

   // 记录链的配置，主要是以太坊经理过几次分叉和提案，为了兼容之前的区块信息
   // 所以做了一些兼容，移植的时候我们只考虑最新版本的内容
   chainConfig *params.ChainConfig
   // chain rules contains the chain rules for the current epoch
   chainRules params.Rules
   // 这个是虚拟机的一些配置参数，是创建解释器的初始化参数，比如所有操作码对应的函数也是在此处配置的
   vmConfig Config
    
   // 解释器对象 它是整个进行虚拟机代码执行的地方。
   interpreters []Interpreter
   interpreter  Interpreter
    
   // 用来终止代码执行
   abort int32
   // callGasTemp holds the gas available for the current call. This is needed because the
   // available gas is calculated in gasCall* according to the 63/64 rule and later
   // applied in opCall*.
   callGasTemp uint64
}
```

#### 3.2.3 NewEVM方法

NewEVM是创建evm的方法

```go
func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
   evm := &EVM{
      Context:      ctx,  // 提供访问当前区块链数据和挖矿环境的函数和数据
      StateDB:      statedb,  // 以太坊状态数据库对象
      vmConfig:     vmConfig,  // 虚拟机配置信息
      chainConfig:  chainConfig, // 当前节点的区块链配置信息
      chainRules:   chainConfig.Rules(ctx.BlockNumber),
      interpreters: make([]Interpreter, 0, 1),
   }

   if chainConfig.IsEWASM(ctx.BlockNumber) {
      // to be implemented by EVM-C and Wagon PRs.
      // 异常捕获
      panic("No supported ewasm interpreter yet.")
   }

   // 这里是重点，通过拓展切片的方法创建解释器，解释器是执行字节码的关键
   evm.interpreters = append(evm.interpreters, NewEVMInterpreter(evm, vmConfig))
   evm.interpreter = evm.interpreters[0]

   return evm
}
```

附：给出vmconfig的数据结构

```go
type Config struct {
   Debug                   bool   // 是否启用调试
   Tracer                  Tracer // Opcode logger
   NoRecursion             bool   // 是否禁止智能合约间的递归调用
   EnablePreimageRecording bool   // Enables recording of SHA3/keccak preimages
   JumpTable [256]operation // evm指令表
   EWASMInterpreter string // External EWASM interpreter options
   EVMInterpreter   string // External EVM interpreter options

   ExtraEips []int // Additional EIPS that are to be enabled
}
```

NewEVMInterpreter方法是创建evm解释器新实例的方法，简单来说虚拟机的一些配置信息vmconfig在这里完善

```go
func NewEVMInterpreter(evm *EVM, cfg Config) *EVMInterpreter {
   // 因为版本迭代，我们需要用STOP指令停止运行，查看解释码是否在版本迭代之后有新的更新，
   if !cfg.JumpTable[STOP].valid {
      var jt JumpTable
      // 通过判断区块链所处状态的不同，更新对应的操作码
      switch {
      case evm.chainRules.IsIstanbul:
         jt = istanbulInstructionSet
      case evm.chainRules.IsConstantinople:
         jt = constantinopleInstructionSet
      case evm.chainRules.IsByzantium:
         jt = byzantiumInstructionSet
      case evm.chainRules.IsEIP158:
         jt = spuriousDragonInstructionSet
      case evm.chainRules.IsEIP150:
         jt = tangerineWhistleInstructionSet
      case evm.chainRules.IsHomestead:
         jt = homesteadInstructionSet
      default:
         jt = frontierInstructionSet
      }
      cfg.JumpTable = jt
   }

   return &EVMInterpreter{
      evm: evm,
      cfg: cfg,
   }
}
```

总之，以太坊在每处理一笔交易时，都会调用NewEVM函数创建EVM对象，哪怕不涉及合约、只是一笔简单的转账。NewEVM的实现也很简单，只是记录相关的参数，同时创建一个解释器对象。Config.JumpTable字段在开始时是无效的，在创建解释器对象时对其进行了填充。

#### 3.3.3 Create方法

Create方法先创建一个合约地址，然后调用更细节的create方法，并返回

```go
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
   // 根据当前合约创建方的账户地址生成一个新的合约地址
   contractAddr = crypto.CreateAddress(caller.Address(), evm.StateDB.GetNonce(caller.Address()))
   return evm.create(caller, &codeAndHash{code: code}, gas, value, contractAddr)
}
```

create方法是创建新智能合约的方法

```go
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *big.Int, address common.Address) ([]byte, common.Address, uint64, error) {
   // 判断evm执行的深度是否超过指定的堆栈最大深度“1024”，报错后直接return
   // 下面是params/protocol_params.go中的定义 
   // CallCreateDepth  uint64 = 1024
   if evm.depth > int(params.CallCreateDepth) {
      return nil, common.Address{}, gas, ErrDepth
   } 
   // 判断合约创建方的账户地址余额是否足够支付gas费用，报错后直接return
   if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
      return nil, common.Address{}, gas, ErrInsufficientBalance
   }
   // 获取当前合约创建方的nonce值，并调用SetNonce方法将nonce+1,传给合约创建方的账户地址
   nonce := evm.StateDB.GetNonce(caller.Address())
   evm.StateDB.SetNonce(caller.Address(), nonce+1)
   // 判断当前合约创建的合约地址是否已经存在合约，报错后直接return
   contractHash := evm.StateDB.GetCodeHash(address)
   if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
      return nil, common.Address{}, 0, ErrContractAddressCollision
   }
   // 调用数据库对象，创建evm虚拟机快照，便于执行过程中产生错误的回滚
   snapshot := evm.StateDB.Snapshot()
   // 根据create函数创建的合约地址，为这个合约地址创建一个账户体系
   evm.StateDB.CreateAccount(address)
   // 当以太坊处于EIP158阶段，调用数据库对象，将合约地址的nonce设置为默认值1
   if evm.chainRules.IsEIP158 {
      evm.StateDB.SetNonce(address, 1)
   }
   // 将交易中附带的value（wei为单位）从账户地址转移到合约地址
   evm.Transfer(evm.StateDB, caller.Address(), address, value)
   // 初始化智能合约，创建新的智能合约对象，注入到合约地址中
   contract := NewContract(caller, AccountRef(address), value, gas)
   // 设置合约的参数，如合约创建者、合约自身地址、合约剩余gas、合约代码和代码的jumpdests记录（jumpdests记录简单理解为合约代码中函数的跳转关系）
   // codeAndHash是creat方法传入的相关参数信息和合约的hash值 
   contract.SetCodeOptionalHash(&address, codeAndHash)
   // 如果以太坊虚拟机被配置成不可递归创建合约，而当前创建合约的过程正是在递归过程中，则直接返回成功，但并没有返回合约代码（第一个返回参数）
   if evm.vmConfig.NoRecursion && evm.depth > 0 {
      return nil, address, gas, nil
   }
   // 如果当前evm状态处于调试状态，则需要将代码放到虚拟机环境进行调试
   if evm.vmConfig.Debug && evm.depth == 0 {
      evm.vmConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.code, gas, value)
   }
   start := time.Now()

   // 将evm对象 合约对象传入run函数开始执行，此函数是核心，等一会分析到Call入口的时候最终也会调用此函数
   ret, err := run(evm, contract, nil, false)
   // 上述函数执行完成后返回的就是我前一章所说的初始化后的合约代码
   // 也就是我们在remix上看到runtime的字节码 以后调用合约代码其实质就是
   // 执行返回后的代码

   // 这里首先需要保证以太坊链阶段是在eip158中，然后保证代码长度不超过指定长度“24576”，将错误值以bool类型赋值给maxCodeSizeExceeded
   maxCodeSizeExceeded := evm.chainRules.IsEIP158 && len(ret) > params.MaxCodeSize
   // 如果合同创建成功运行并且没有返回错误，就计算存储代码所需的费用。 
   // 如果由于气体不足而无法存储该代码，就设置一个错误，并通过下面的错误检查条件进行处理。
   if err == nil && !maxCodeSizeExceeded {
      // 根据代码长度计算存储费用
      // CreateDataGas = 200 wei
      createDataGas := uint64(len(ret)) * params.CreateDataGas
      // 如果实际消耗gas值小于预先扣除gas值，将剩余值退回给账户，设置合约到以太坊数据库
      if contract.UseGas(createDataGas) {
         evm.StateDB.SetCode(address, ret)
      } else {
         err = ErrCodeStoreOutOfGas
      }
   }

   // 当合约代码长度超过限制或报错时显示以太坊链状态处于IsHomestead阶段或报错显示非ErrCodeStoreOutOfGas的错误，我们将返回快照并消耗掉所有剩余的气体。
   if maxCodeSizeExceeded || (err != nil && (evm.chainRules.IsHomestead || err != ErrCodeStoreOutOfGas)) {
      // 更改以太坊状态数据库至快照的状态
      evm.StateDB.RevertToSnapshot(snapshot)
      // 如果没有发生快照的恢复执行错误，不给相应账户退回任何gas值
      if err != errExecutionReverted {
         contract.UseGas(contract.Gas)
      }
   }
   // 这里将代码容量过载的问题转化为错误
   if maxCodeSizeExceeded && err == nil {
      err = errMaxCodeSizeExceeded
   }
   if evm.vmConfig.Debug && evm.depth == 0 {
      evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
   }
   return ret, address, contract.Gas, err

}
```

附：UseGas方法

```go
func (c *Contract) UseGas(gas uint64) (ok bool) {
   if c.Gas < gas {
      return false
   }
   c.Gas -= gas
   return true
}
```

#### 3.3.4 run方法

run方法真正运行智能合约的方法，利用字节码解释器进行预编译，在将来要说的call方法中也将调用

```go
func run(evm *EVM, contract *Contract, input []byte, readOnly bool) ([]byte, error) {
   // 如果合约账户地址不为空
   if contract.CodeAddr != nil {
      // 设置为适用于Frontier和Homestead阶段预编译以太坊集
      precompiles := PrecompiledContractsHomestead
      // 下面设置是拜占庭阶段的预编译以太坊集
      if evm.chainRules.IsByzantium {
         precompiles = PrecompiledContractsByzantium
      }
      // 下面是伊斯坦丁堡阶段的预编译以太坊集
      if evm.chainRules.IsIstanbul {
         precompiles = PrecompiledContractsIstanbul
      }
      // 如果是上述阶段的特殊地址，RunPrecompiledContract调用其阶段对应的Run方法返回的字节码结果，并输出
      if p := precompiles[*contract.CodeAddr]; p != nil {
         return RunPrecompiledContract(p, input, contract)
      }
   }
   for _, interpreter := range evm.interpreters {
       // 循环evm解释器集，找到适合的解释器
      if interpreter.CanRun(contract.Code) {
         if evm.interpreter != interpreter {
            // 这里设置把最适合的解释器放在解释器集第一个位置，便于后面的回滚
            defer func(i Interpreter) {
               evm.interpreter = i
            }(evm.interpreter)
            evm.interpreter = interpreter
         }
         // 通过适合的解释器执行编译，返回字节码
         return interpreter.Run(contract, input, readOnly)
      }
   }
   return nil, ErrNoCompatibleInterpreter
}
```

下面是Run方法，这才是真正的合约编译地点，通过适合的解释器运行，但是可以发现适合的解释器仅有EVMInterpreter，所以下面是函数的整体结构

1. Run循环，从第0个字节开始，并使用给定的输入数据评估合同的代码，并返回返回字节片，如果发生错误则返回错误。
2. 重要的是要注意，解释器返回的任何错误都应被视为“还原并消耗所有gas”操作，但errExecutionReverted除外，这意味着“还原并保留gas”。

```go
func (in *EVMInterpreter) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
   if in.intPool == nil {
      in.intPool = poolOfIntPools.get()
      defer func() {
         poolOfIntPools.put(in.intPool)
         in.intPool = nil
      }()
   }
   // Increment the call depth which is restricted to 1024
   in.evm.depth++
   defer func() { in.evm.depth-- }()

   // Make sure the readOnly is only set if we aren't in readOnly yet.
   // This makes also sure that the readOnly flag isn't removed for child calls.
   if readOnly && !in.readOnly {
      in.readOnly = true
      defer func() { in.readOnly = false }()
   }

   // Reset the previous call's return data. It's unimportant to preserve the old buffer
   // as every returning call will return new data anyway.
   in.returnData = nil

   // Don't bother with the execution if there's no code.
   if len(contract.Code) == 0 {
      return nil, nil
   }
   // 下面这些变量应该说满足了一个字节码执行的所有条件
   // 有操作码 内存 栈 PC计数器 
   // 强烈建议使用debug工具去跟踪一遍执行的流程 
   // 其实它的执行流程就和上一章我们人肉执行的流程一样
   var (
      op    OpCode        // current opcode
      mem   = NewMemory() // bound memory
      stack = newstack()  // local stack
      // For optimisation reason we're using uint64 as the program counter.
      // It's theoretically possible to go above 2^64. The YP defines the PC
      // to be uint256. Practically much less so feasible.
      pc   = uint64(0) // program counter
      cost uint64
      // copies used by tracer
      pcCopy  uint64 // needed for the deferred Tracer
      gasCopy uint64 // for Tracer to log gas remaining before execution
      logged  bool   // deferred Tracer should ignore already logged steps
      res     []byte // result of the opcode execution function
   )
   contract.Input = input

   // Reclaim the stack as an int pool when the execution stops
   defer func() { in.intPool.put(stack.data...) }()

   if in.cfg.Debug {
      defer func() {
         if err != nil {
            if !logged {
               in.cfg.Tracer.CaptureState(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
            } else {
               in.cfg.Tracer.CaptureFault(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
            }
         }
      }()
   }
   // The Interpreter main run loop (contextual). This loop runs until either an
   // explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
   // the execution of one of the operations or until the done flag is set by the
   // parent context.
   // 开始循环PC计数执行 直到有中止执行或者跳出循环
   for atomic.LoadInt32(&in.evm.abort) == 0 {
      if in.cfg.Debug {
         // Capture pre-execution values for tracing.
         logged, pcCopy, gasCopy = false, pc, contract.Gas
      }
      // Get the operation from the jump table and validate the stack to ensure there are
      // enough stack items available to perform the operation.
      // 根据PC计数器获取操作码
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
      // Static portion of gas
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
      // 计算此操作花费的gas数量
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
   return nil, nil
}
```

到了这里整个部署合约流程就完成了, 部署合约时是从evm.Create->run->interper.run 然后在执行codeCopy指令后把runtime的内容返回出来。 在evm.Create函数中我们也看到了当run执行完成后会把runtime的合约代码最终设置到合约地址名下。 整个合约部署就算完成了。

#### 3.3.5 Call方法

分析完合约创建接着就该分析合约调用代码了。 调用智能合约和部署在以太坊交易上看来就是to的地址不在是nil而是一个具体的合约地址了。 同时input的内容不再是整个合约编译后的字节码了而是调用函数和对应的实参组合的内容。 这里就涉及到另一个东西那就是abi的概念。 abi描述了整个接口的详细信息， 根据abi可以解包和打包input调用的数据。









