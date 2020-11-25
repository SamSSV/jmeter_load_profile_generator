package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const loadStepPattern = `            <collectionProp name="%d">
		<stringProp name="%d">%d</stringProp>
		<stringProp name="%d">%d</stringProp>
		<stringProp name="%d">%d</stringProp>
	    </collectionProp>`

func readFileContent(filePath string) ([]byte, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading file [%s]: %w", filePath, err)
	}

	return content, nil
}

func overwriteFileContent(filePath string, newContent []byte) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed opening file: %w", err)
	}
	defer file.Close()

	len, err := file.Write(newContent)
	if err != nil {
		return fmt.Errorf("failed writing to file: %w", err)
	}
	fmt.Printf("\nLength: %d bytes\n", len)
	fmt.Printf("File Name: %s\n", file.Name())

	return nil
}

func createLoadProfile(l loadProfile) string {
	var output strings.Builder
	for i := 0; i < l.numSteps; i++ {
		var prev int
		var next int
		if i == 0 {
			prev = l.initLoad
			next = l.initLoad + l.increment
		} else {
			prev = l.initLoad + (l.increment * i)
			next = prev + l.increment
		}
		output.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), prev, prev, next, next, l.rampUpPropName, l.rampUp))
		output.WriteString("\n")
		output.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), next, next, next, next, l.stepDurationPropName, l.stepDuration))
		output.WriteString("\n")
	}
	output.WriteString(`          </collectionProp>\n      	</kg`)

	return output.String()
}

func printHelp() {
	fmt.Println("set next args: jmxPath, initLoad,increment,rampUp,stepDuration,numSteps")
	fmt.Println("Args example: path/to/test.jmx 10,10,60,180,3")
	fmt.Println("initLoad,increment,rampUp,stepDuration,numSteps values must be gt 0")
}

func getInputArgs() (string, []string, error) {
	const profileVarsAmount = 5
	var numericPattern = regexp.MustCompile(`^\d+$`)

	if len(os.Args) <= 2 {
		printHelp()

		return "", nil, errors.New("Invalid input arguments")
	}

	path := os.Args[1]
	values := strings.Split(os.Args[2], ",")

	// validate input profile values
	if len(values) != profileVarsAmount {
		printHelp()

		return "", nil, fmt.Errorf("Invalid amount of profile values: %d, %d required", len(values), profileVarsAmount)
	}
	for _, val := range values {
		if !numericPattern.MatchString(val) || val == "0" {
			printHelp()

			return "", nil, errors.New("Invalid input data for profile values")
		}
	}

	return path, values, nil
}

func assignLoadProfileParams(values []string, lp *loadProfile) {
	lp.initLoad, _ = strconv.Atoi(values[0])
	lp.increment, _ = strconv.Atoi(values[1])
	lp.rampUp, _ = strconv.Atoi(values[2])
	lp.rampUpPropName = lp.rampUpPropName + lp.rampUp
	lp.stepDuration, _ = strconv.Atoi(values[3])
	lp.stepDurationPropName = lp.stepDurationPropName + lp.stepDuration
	lp.numSteps, _ = strconv.Atoi(values[4])
	lp.totalTestDuration = lp.numSteps*(lp.rampUp+lp.stepDuration) + 5
}

type loadProfile struct {
	jmxPath              string
	initLoad             int
	increment            int
	rampUp               int
	stepDuration         int
	numSteps             int
	rampUpPropName       int
	stepDurationPropName int
	totalTestDuration    int
}

func main() {

	path, values, err := getInputArgs()
	if err != nil {
		log.Fatalln(err)
	}

	loadP := loadProfile{jmxPath: path}

	assignLoadProfileParams(values, &loadP)

	fmt.Println(loadP.jmxPath)

	oldContent, err := readFileContent(loadP.jmxPath)
	if err != nil {
		log.Fatalln(err)
	}

	newLoadProfile := createLoadProfile(loadP)
	loadProfMatcher := regexp.MustCompile(`\<collectionProp name="[^load_profile]*"\>(\s*|.*)*\<\/kg`) // `<collectionProp name="[^load_profile]">(.|\s)*?<\/kg`

	newContent := loadProfMatcher.ReplaceAllLiteral(oldContent, []byte(newLoadProfile))

	loadProfDurationPattern := `<stringProp name="Hold">%d</stringProp>`
	loadProfDurationMatcher := regexp.MustCompile(`<stringProp name="Hold">(\d*?)<\/stringProp>`)

	newContent = loadProfDurationMatcher.ReplaceAllLiteral(newContent, []byte(fmt.Sprintf(loadProfDurationPattern, loadP.totalTestDuration)))
	//fmt.Println(newContent)
	err = overwriteFileContent(loadP.jmxPath, newContent)
	if err != nil {
		log.Fatalln(err)
	}
}
