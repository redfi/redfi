package redfi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectRule(t *testing.T) {
	p := &Plan{
		Rules: []*Rule{},
	}

	// // test ip matching
	p.Rules = append(p.Rules, &Rule{
		Delay:      1e3,
		ClientAddr: "192.0.0.1:8001",
	})

	rule := p.SelectRule("192.0.0.1", []byte(""))
	if rule == nil {
		t.Fatal("rule must not be nil")
	}

	// test command matching
	p.Rules = []*Rule{}
	p.Rules = append(p.Rules, &Rule{
		Delay:   1e3,
		Command: "GET",
	})
	p.MarshalCommands()

	rule = p.SelectRule("192.0.0.1", []byte("\r\nGET\r\nfff"))
	if rule == nil {
		t.Fatal("rule must not be nil")
	}

	rule = p.SelectRule("172.0.0.1", []byte("\r\nKEYS\r\nfff"))
	if rule != nil {
		fmt.Println(rule)
		t.Fatal("rule must BE nil")
	}

}

func TestAddDeleteGetRule(t *testing.T) {
	p := NewPlan()

	r := Rule{
		Name:       "clients_delay",
		Delay:      50,
		Percentage: 20,
	}
	p.AddRule(r)

	if len(p.Rules) != 1 {
		t.Fatal("rule wasn't added")
	}
	if !(p.Rules[0].Delay == r.Delay && p.Rules[0].Percentage == r.Percentage) {
		t.Fatal("rule added doesn't match")
	}

	fetchedRule, err := p.GetRule("clients_delay")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(fetchedRule)

	err = p.DeleteRule("clients_delay")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.GetRule("clients_delay")
	if err == nil {
		t.Fatal(err)
	}
	fmt.Println(fetchedRule)

}

func TestCommandFaultInjection(t *testing.T) {
	var testCases = []struct {
		description  string
		clientAddr   string
		rule         Rule
		redisCommand []byte
		expectedRule Rule
	}{
		{
			description: "success case: match command (upper case) and return the expected rule",
			clientAddr:  "192.0.0.1",
			rule: Rule{
				Name:       "Invalid Key",
				ReturnErr:  "ERR_INVALID_KEY",
				Command:    "GET",
				Percentage: 100,
			},
			redisCommand: []byte("\r\nGET\r\nkey1"),
			expectedRule: Rule{
				Name:         "Invalid Key",
				ReturnErr:    "ERR_INVALID_KEY",
				Command:      "GET",
				Percentage:   100,
				marshaledCmd: marshalCommand("GET"),
				hits:         1,
			},
		},
		{
			description: "success case: match command (lower case) and return the expected rule",
			clientAddr:  "192.0.0.1",
			rule: Rule{
				Name:       "Inject delay",
				Delay:      5000,
				Command:    "GET",
				Percentage: 100,
			},
			redisCommand: []byte("\r\nget\r\nkey1"),
			expectedRule: Rule{
				Name:         "Inject delay",
				Delay:        5000,
				Command:      "GET",
				Percentage:   100,
				marshaledCmd: marshalCommand("GET"),
				hits:         1,
			},
		},
	}

	for _, testCase := range testCases {
		t.Logf("running test case: %v", testCase.description)

		plan := NewPlan()

		err := plan.AddRule(testCase.rule)
		if err != nil {
			t.Fatalf("error adding rule: %v", err.Error())
		}

		actualRule := plan.SelectRule(testCase.clientAddr, testCase.redisCommand)
		assert.Equal(t, *actualRule, testCase.expectedRule)
	}
}
