package azbp010

func badConstantReturn() string {
	const str = "TEST" // want `AZBP010`
	return str
}

func badVariableReturn() int {
	var num = 42 // want `AZBP010`
	return num
}

func badAssignmentReturn() string {
	result := "hello" // want `AZBP010`
	return result
}

func badComplexAssignmentReturn() error {
	err := someFunction() // want `AZBP010`
	return err
}

func goodPartialReturn() string {
	str, _ := someFunction2()
	return str
}

func goodMultipleVariables() (string, int) {
	str := "TEST"
	num := 42
	return str, num
}

func goodDifferentOrder() (int, string) {
	a, b := someFunction3()
	return b, a
}

func goodDuplicateReturn() int {
	a, _ := someFunction4()
	return a
}

func goodMultipleInOrder() (string, error) {
	result, err := someFunction2() // want 	`AZBP010`
	return result, err
}

func someFunction() error {
	return nil
}

func someFunction2() (string, error) {
	return "", nil
}

func someFunction3() (string, int) {
	return "test", 42
}

func someFunction4() (int, int) {
	return 1, 2
}
