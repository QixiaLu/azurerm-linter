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

func test1() string {
	str, _ := someFunction2()
	return str
}

func goodMultipleVariables() (string, int) {
	str := "TEST"
	num := 42
	return str, num
}

func someFunction() error {
	return nil
}

func someFunction2() (string, error) {
	return "", nil
}
