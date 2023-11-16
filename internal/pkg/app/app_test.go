package app

import "testing"

func testchangingIndex(t *testing.T) {
	testData := [][]string{{"name","age","_index"},{"Frank","25","1"},{"John","65","2"},{"Lisa","32","3"}}
	
	outputData := changingIndex(testData)

	expected1 := "i1"
	expected2 := "i2"

	if outputData[1][2] != expected1 {
		t.Errorf("%s != %s", outputData[1][2], expected1)
	}
	if outputData[2][2] != expected2 {
		t.Errorf("%s != %s", outputData[2][2], expected2)
	}
}