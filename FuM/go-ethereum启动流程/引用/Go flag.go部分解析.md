```go
@author FuMing
@data 2020.03.12
```

# Go flag.go部分解析

[TOC]



### Parse函数

此函数将较长的终端命令转化为命令-参数键值，主要依赖```parseOne```函数。

```go
// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		//f.parseOne()就是解析f中的第一个参数。每次解析过的参数都会被存入f.actual后删除，这样每次执行f.parseOne()都保证被解析的参数未被解析过。
		seen, err := f.parseOne()//seen为是否发现参数，发现并转换成功则为true，否则为false
		if seen {
			continue //需要一直把命令解析完
		}
		if err == nil {
			break //参数正确解析完会执行此
		}
		//能执行至此说明seen==false&&err != nil,说明出现错误，下面switch处理错误
		switch f.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}
```

### parseOne函数

此函数将长的终端命令中的第一个命令转化为命令-参数键值。

```go
// parseOne parses one flag. It reports whether a flag was seen.
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	//没有长度为0或1的flag，也没有第一位不是"-"的flag
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	//运行到此，s[0]必为"-"，故numMinuses从1开始
	numMinuses := 1  //numMinuses是一个flag中"-"的个数
	//如果命令由"--"开头
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]// 舍弃此参数
			return false, nil
		}
	}
	name := s[numMinuses:]//拿到flag除"-"以外的部分，也就是flag名
	//下面都是无法解析的flag格式
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s)
	}
	//确定是一个flag,接下来检查一下后面有没有参数
	// it's a flag. does it have an argument?
	f.args = f.args[1:]//舍弃第一个参数
	hasValue := false//目前hasValue未知，设为false
	value := ""
	for i := 1; i < len(name); i++ { // 等号不能在第一位，故i从1始
		//发现等号
		if name[i] == '=' {
			value = name[i+1:]//获取等号后的值，即此flag的参数
			hasValue = true
			name = name[0:i]
			break
			//name和value被分开
		}
	}
	m := f.formal // m:map[string]*flag.Flag
	//从flag映射中拿到flag并检测命令是否存在
	flag, alreadythere := m[name] // BUG
	if !alreadythere {
		if name == "help" || name == "h" { // special case for nice help message.
			f.usage()
			return false, ErrHelp
		}
		return false, f.failf("flag provided but not defined: -%s", name)//命令不存在
	}

	if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() { // special case: doesn't need an arg
		if hasValue {
			if err := fv.Set(value); err != nil {
				return false, f.failf("invalid boolean value %q for -%s: %v", value, name, err)
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return false, f.failf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		if !hasValue && len(f.args) > 0 {
			// value is the next arg
			hasValue = true
			value, f.args = f.args[0], f.args[1:]
		}
		if !hasValue {
			return false, f.failf("flag needs an argument: -%s", name)
		}
		if err := flag.Value.Set(value); err != nil {
			return false, f.failf("invalid value %q for flag -%s: %v", value, name, err)
		}
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag //存储对参数的分解
	return true, nil
}
```

