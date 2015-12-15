package teamcity

import "testing"

func TestClientGetBuildProperties(t *testing.T) {
	client := NewTestClient(newResponse(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<properties count="3">
  <property name="build.counter" value="12"/>
  <property name="build.number" value="supermama123"/>
  <property name="env.BUILD_NUMBER" value="supermama123"/>
</properties>`), nil)

	props, err := client.GetBuildProperties("999999")

	if len(props) != 3 {
		t.Fatal("Expected to have 3 properties, found", len(props))
	}

	if err != nil {
		t.Fatal("Expected no error, got", err)
	}
}
