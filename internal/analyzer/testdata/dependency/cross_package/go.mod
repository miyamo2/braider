module example.com/dependency/cross_package

go 1.24

replace github.com/miyamo2/braider/pkg => ./../../../../../pkg

require github.com/miyamo2/braider/pkg v0.0.0-00010101000000-000000000000
