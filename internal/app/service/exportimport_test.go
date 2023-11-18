package service

import "testing"

func TestChangingIndex(t *testing.T) {
	testData := [][]string{{"name","age","_index"},{"Frank","25","1"},{"John","65","2"},{"Lisa","32","3"}}
	
	outputData, err := changingIndex(testData, 1528, 2)

	expected1 := "1527"
	expected2 := "1528"
	expected3 := "1529"

	if err != nil {
		t.Error(err)
	}

	if outputData[1][2] != expected1 {
		t.Errorf("%s != %s", outputData[1][2], expected1)
	}
	if outputData[2][2] != expected2 {
		t.Errorf("%s != %s", outputData[2][2], expected2)
	}
	if outputData[3][2] != expected3 {
		t.Errorf("%s != %s", outputData[3][2], expected2)
	}

	outputData, err = changingIndex(testData, 1528, 2)


	if err != nil {
		t.Error(err)
	}

	if outputData[1][2] != expected1 {
		t.Errorf("%s != %s", outputData[1][2], expected1)
	}
	if outputData[2][2] != expected2 {
		t.Errorf("%s != %s", outputData[2][2], expected2)
	}
	if outputData[3][2] != expected3 {
		t.Errorf("%s != %s", outputData[3][2], expected2)
	}


}

func TestGetStringNumber(t *testing.T) {
	number1 := getStringNumber("kobo!A1:XYZ")
	number2 := getStringNumber("kobo2!A27765:XYZ")

	if number1 != 1 {
		t.Errorf("number1 != 1, but %d", number1)
	}
	if number2 != 27765 {
		t.Errorf("number2 != 27765, but %d", number1)
	}
}