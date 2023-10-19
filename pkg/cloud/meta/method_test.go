package meta

import (
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/api/compute/v1"
)

// structs below simulate the shape of the Compute APIs for testing Method.

type opCall struct{}

func (*opCall) Do() (*compute.Operation, error) { return nil, nil }

type pagesItem struct{}

type pagesResult struct {
	Items []*pagesItem
}

type pagesCall struct{}

func (*pagesCall) Do() (*pagesResult, error)    { return nil, nil }
func (*pagesCall) Pages() (*pagesResult, error) { return nil, nil }

type getCall struct{}

func (*getCall) Do() (*int, error) { return nil, nil }

type fakeService struct{}

func (*fakeService) GlobalOperation(string, string, int) *opCall           { return nil }
func (*fakeService) GlobalPages(string, string, int) *pagesCall            { return nil }
func (*fakeService) GlobalGet(string, string, int) *getCall                { return nil }
func (*fakeService) RegionalOperation(string, string, string, int) *opCall { return nil }
func (*fakeService) RegionalPages(string, string, string, int) *pagesCall  { return nil }
func (*fakeService) RegionalGet(string, string, string, int) *getCall      { return nil }
func (*fakeService) ZonalOperation(string, string, string, int) *opCall    { return nil }
func (*fakeService) ZonalPages(string, string, string, int) *pagesCall     { return nil }
func (*fakeService) ZonalGet(string, string, string, int) *getCall         { return nil }

func TestMethod(t *testing.T) {
	methodOrDie := func(name string) reflect.Method {
		reflectm, ok := reflect.TypeOf(&fakeService{}).MethodByName(name)
		if !ok {
			panic(fmt.Sprintf("Method %q not in FakeService", name))
		}
		return reflectm
	}

	for _, tc := range []struct {
		name string
		kt   KeyType
		m    reflect.Method

		wantKind          MethodKind
		wantHook          string
		wantFcnArgs       string
		wantInterfaceFunc string
	}{
		{
			name:              "global operation",
			kt:                Global,
			m:                 methodOrDie("GlobalOperation"),
			wantKind:          MethodOperation,
			wantHook:          "GlobalOperationHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) error",
			wantFcnArgs:       "GlobalOperation(ctx context.Context, key *meta.Key, arg0 int, options ...Option) error",
			wantInterfaceFunc: "GlobalOperation(context.Context, *meta.Key, int, ...Option) error",
		},
		{
			name:              "global pages",
			kt:                Global,
			m:                 methodOrDie("GlobalPages"),
			wantKind:          MethodPaged,
			wantHook:          "GlobalPagesHook func(context.Context, *meta.Key, int, *filter.F, *MockFakes, ...Option) ([]*ga.pagesItem, error)",
			wantFcnArgs:       "GlobalPages(ctx context.Context, key *meta.Key, arg0 int, fl *filter.F, options ...Option) ([]*ga.pagesItem, error)",
			wantInterfaceFunc: "GlobalPages(context.Context, *meta.Key, int, *filter.F, ...Option) ([]*ga.pagesItem, error)",
		},
		{
			name:              "global get",
			kt:                Global,
			m:                 methodOrDie("GlobalGet"),
			wantKind:          MethodGet,
			wantHook:          "GlobalGetHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) (*ga.int, error)",
			wantFcnArgs:       "GlobalGet(ctx context.Context, key *meta.Key, arg0 int, options ...Option) (*ga.int, error)",
			wantInterfaceFunc: "GlobalGet(context.Context, *meta.Key, int, ...Option) (*ga.int, error)",
		},
		{
			name:              "regional operation",
			kt:                Regional,
			m:                 methodOrDie("RegionalOperation"),
			wantKind:          MethodOperation,
			wantHook:          "RegionalOperationHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) error",
			wantFcnArgs:       "RegionalOperation(ctx context.Context, key *meta.Key, arg0 int, options ...Option) error",
			wantInterfaceFunc: "RegionalOperation(context.Context, *meta.Key, int, ...Option) error",
		},
		{
			name:              "regional pages",
			kt:                Regional,
			m:                 methodOrDie("RegionalPages"),
			wantKind:          MethodPaged,
			wantHook:          "RegionalPagesHook func(context.Context, *meta.Key, int, *filter.F, *MockFakes, ...Option) ([]*ga.pagesItem, error)",
			wantFcnArgs:       "RegionalPages(ctx context.Context, key *meta.Key, arg0 int, fl *filter.F, options ...Option) ([]*ga.pagesItem, error)",
			wantInterfaceFunc: "RegionalPages(context.Context, *meta.Key, int, *filter.F, ...Option) ([]*ga.pagesItem, error)",
		},
		{
			name:              "regional get",
			kt:                Regional,
			m:                 methodOrDie("RegionalGet"),
			wantKind:          MethodGet,
			wantHook:          "RegionalGetHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) (*ga.int, error)",
			wantFcnArgs:       "RegionalGet(ctx context.Context, key *meta.Key, arg0 int, options ...Option) (*ga.int, error)",
			wantInterfaceFunc: "RegionalGet(context.Context, *meta.Key, int, ...Option) (*ga.int, error)",
		},
		{
			name:              "zonal operation",
			kt:                Zonal,
			m:                 methodOrDie("ZonalOperation"),
			wantKind:          MethodOperation,
			wantHook:          "ZonalOperationHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) error",
			wantFcnArgs:       "ZonalOperation(ctx context.Context, key *meta.Key, arg0 int, options ...Option) error",
			wantInterfaceFunc: "ZonalOperation(context.Context, *meta.Key, int, ...Option) error",
		},
		{
			name:              "zonal pages",
			kt:                Zonal,
			m:                 methodOrDie("ZonalPages"),
			wantKind:          MethodPaged,
			wantHook:          "ZonalPagesHook func(context.Context, *meta.Key, int, *filter.F, *MockFakes, ...Option) ([]*ga.pagesItem, error)",
			wantFcnArgs:       "ZonalPages(ctx context.Context, key *meta.Key, arg0 int, fl *filter.F, options ...Option) ([]*ga.pagesItem, error)",
			wantInterfaceFunc: "ZonalPages(context.Context, *meta.Key, int, *filter.F, ...Option) ([]*ga.pagesItem, error)",
		},
		{
			name:              "zonal get",
			kt:                Zonal,
			m:                 methodOrDie("ZonalGet"),
			wantKind:          MethodGet,
			wantHook:          "ZonalGetHook func(context.Context, *meta.Key, int, *MockFakes, ...Option) (*ga.int, error)",
			wantFcnArgs:       "ZonalGet(ctx context.Context, key *meta.Key, arg0 int, options ...Option) (*ga.int, error)",
			wantInterfaceFunc: "ZonalGet(context.Context, *meta.Key, int, ...Option) (*ga.int, error)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			si := &ServiceInfo{
				Object:            "Fake",
				Service:           "Fakes",
				Resource:          "fakes",
				keyType:           tc.kt,
				serviceType:       reflect.TypeOf(&fakeService{}),
				additionalMethods: []string{tc.m.Name},
			}
			method := newMethod(si, tc.m)

			if method.kind != tc.wantKind {
				t.Errorf("method.kind = %d, want %d", method.kind, tc.wantKind)
			}
			if method.MockHook() != tc.wantHook {
				t.Errorf("MockHook() = %q, want %q", method.MockHook(), tc.wantHook)
			}
			if method.FcnArgs() != tc.wantFcnArgs {
				t.Errorf("FcnArgs() = %q, want %q", method.FcnArgs(), tc.wantFcnArgs)
			}
			if method.InterfaceFunc() != tc.wantInterfaceFunc {
				t.Errorf("InterfaceFunc() = %q, want %q", method.InterfaceFunc(), tc.wantInterfaceFunc)
			}
		})
	}
}
