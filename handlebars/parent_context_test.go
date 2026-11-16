// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import "testing"

func TestIssue1173ParentContext(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"default this", "{{name}} {{> myPartial this}}", object("name", "root"), "root root", nil, map[string]string{"myPartial": "{{../name}}"}, nil},
		{"normal parent navigation", "{{#with child}}{{childValue}} {{../rootValue}}{{/with}}", object("rootValue", "fromRoot", "child", object("childValue", "fromChild")), "fromChild fromRoot", nil, nil, nil},
		{"implicit this", "{{name}} {{> myPartial}}", object("name", "root", "value", "rootValue"), "root root", nil, map[string]string{"myPartial": "{{name}}"}, nil},
		{"explicit this", "{{name}} {{> myPartial this}}", object("name", "root", "value", "rootValue"), "root root", nil, map[string]string{"myPartial": "{{name}}"}, nil},
		{"single parent", "{{rootValue}} {{#with child}}{{childValue}} {{> myPartial ..}}{{/with}}", object("rootValue", "fromRoot", "child", object("childValue", "fromChild")), "fromRoot fromChild fromRoot", nil, map[string]string{"myPartial": "{{rootValue}}"}, nil},
		{"multi parent", "{{rootValue}} {{#with level1}}{{#with level2}}{{level2Value}} {{> myPartial ../..}}{{/with}}{{/with}}", object("rootValue", "fromRoot", "level1", object("level1Value", "fromLevel1"), "level2", object("level2Value", "fromLevel2")), "fromRoot fromLevel2 fromRoot", nil, map[string]string{"myPartial": "{{rootValue}}"}, nil},
		{"nested partials", "{{rootValue}} {{> myPartial}}", object("rootValue", "fromRoot", "nestedValue", "fromNested"), "fromRoot fromRoot fromNested", nil, map[string]string{"myPartial": "{{rootValue}} {{> anotherPartial this}}", "anotherPartial": "{{nestedValue}}"}, nil},
		{"complex nesting", "{{root}} {{#with outer}}{{outer}} {{#with inner}}{{inner}} {{> myPartial ../..}}{{/with}}{{/with}}", object("root", "ROOT", "outer", object("outer", "OUTER"), "inner", object("inner", "INNER")), "ROOT OUTER INNER ROOT", nil, map[string]string{"myPartial": "{{root}}"}, nil},
		{"parent properties", "{{title}} {{#with content}}{{body}} {{> myPartial ..}}{{/with}}", object("title", "MyTitle", "content", object("body", "MyBody"), "footer", "MyFooter"), "MyTitle MyBody MyTitle:MyFooter", nil, map[string]string{"myPartial": "{{title}}:{{footer}}"}, nil},
		{"three levels", "{{topLevel}} {{#with level1}}{{#with level2}}{{#with level3}}{{level3Value}} {{> myPartial ../../..}}{{/with}}{{/with}}{{/with}}", object("topLevel", "TOP", "level1", object("level1Value", "fromLevel1"), "level2", object("level2Value", "fromLevel2"), "level3", object("level3Value", "fromLevel3")), "TOP fromLevel3 TOP", nil, map[string]string{"myPartial": "{{topLevel}}"}, nil},
		{"normal explicit context", "{{rootName}} {{> myPartial myContext}}", object("rootName", "rootValue", "myContext", object("name", "myContextValue")), "rootValue myContextValue", nil, map[string]string{"myPartial": "{{name}}"}, nil},
		{"two level access", "{{grandParent}} {{#with parent}}{{parent}} {{#with child}}{{child}} {{> myPartial ../..}}{{/with}}{{/with}}", object("grandParent", "GRAND", "parent", object("parent", "PARENT"), "child", object("child", "CHILD")), "GRAND PARENT CHILD GRAND", nil, map[string]string{"myPartial": "{{grandParent}}"}, nil},
	})
}
