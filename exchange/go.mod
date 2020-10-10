module echo-demo

go 1.14

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/urfave/cli v1.22.4
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 // indirect
)

replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190829043050-9756ffdc2472
	golang.org/x/net v0.0.0-20181023162649-9b4f9f5ad519 => github.com/golang/net v0.0.0-20181023162649-9b4f9f5ad519
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3 => github.com/golang/net v0.0.0-20181220203305-927f97764cc3
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3 => github.com/golang/net v0.0.0-20190404232315-eb5bcb51f2a3
	golang.org/x/sys => github.com/golang/sys v0.0.0-20190830142957-1e83adbbebd0
	golang.org/x/text v0.3.0 => github.com/golang/text v0.3.0
	golang.org/x/tools v0.0.0-20181221001348-537d06c36207 => github.com/golang/tools v0.0.0-20181221001348-537d06c36207
)
