#### Big包：整数高精度计算

 实际开发中，对于超出 int64 或者 uint64 类型的大数进行计算时，如果对精度没有要求，使用 float32 或者 float64 就可以胜任，但如果对精度有严格要求的时候，我们就不能使用浮点数了，因为浮点数在内存中只能被近似的表示。

Go语言中 math/big 包实现了大数字的多精度计算，支持 Int（有符号整数）、Rat（有理数）和 Float（浮点数）等数字类型。

这些类型可以实现任意位数的数字，只要内存足够大，但缺点是需要更大的内存和处理开销，这使得它们使用起来要比内置的数字类型慢很多。 

常用定义：

```go
type Int struct {
	neg bool // sign
	abs nat  // absolute value of the integer
}

//newInt函数只支持输入int64类型整数值返回big.Int，如果不是int64类型（如int类型），需要转换成int64再进行运算
func NewInt(x int64) *Int {
	return new(Int).SetInt64(x)
}

func (z *Int) Set(x *Int) *Int {
	if z != x {
		z.abs = z.abs.set(x.abs)
		z.neg = x.neg
	}
	return z
}

func (z *Int) Abs(x *Int) *Int {
	z.Set(x)
	z.neg = false
	return z
}

// Add sets z to the sum x+y and returns z.
func (z *Int) Add(x, y *Int) *Int {
	neg := x.neg
	if x.neg == y.neg {
		// x + y == x + y
		// (-x) + (-y) == -(x + y)
		z.abs = z.abs.add(x.abs, y.abs)
	} else {
		// x + (-y) == x - y == -(y - x)
		// (-x) + y == y - x == -(x - y)
		if x.abs.cmp(y.abs) >= 0 {
			z.abs = z.abs.sub(x.abs, y.abs)
		} else {
			neg = !neg
			z.abs = z.abs.sub(y.abs, x.abs)
		}
	}
	z.neg = len(z.abs) > 0 && neg // 0 has no sign
	return z
}
```

关于int64,int32,int16,uint的补充说明：

Int16  意思是16位整数(16bit integer)，相当于short  占2个字节   -32768 ~ 32767

Int32  意思是32位整数(32bit integer), 相当于 int      占4个字节   -2147483648 ~ 2147483647

Int64  意思是64位整数(64bit interger), 相当于 long long   占8个字节   -9223372036854775808 ~ 9223372036854775807

Byte  相当于byte(unsigned char)   0 ~ 255

WORD 等于  unsigned short     0 ~ 65535

uint则是不带符号的，表示范围是：2^32即0到4294967295。

#### 位运算

位运算涉及到底层优化，一些算法及源码可能会经常遇见。

常用的位运算:

> > ```
> >   &      与 AND
> > 
> >   |      或OR
> > 
> >   ^      异或XOR，一元运算表示按位取反
> > 
> >   &^     位清空 (AND NOT)
> > 
> >   <<     左移
> > 
> >   >>	 右移
> > ```

#### 关于Byte

 在go里面，byte是uint8的别名 。