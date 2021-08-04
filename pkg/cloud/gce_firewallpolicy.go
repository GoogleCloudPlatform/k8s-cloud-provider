package cloud

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	alpha "google.golang.org/api/compute/v0.alpha"
	"k8s.io/klog/v2"
)

type AlphaNetworkFirewallPoliciesOps interface {
	GetRule(context.Context, *meta.Key, *int64) (*alpha.FirewallPolicyRule, error)
	RemoveRule(context.Context, *meta.Key, *int64) error
}

// GetRule is a method on GCEAlphaNetworkFirewallPolicies.
func (g *GCEAlphaNetworkFirewallPolicies) GetRule(ctx context.Context, key *meta.Key, priority *int64) (*alpha.FirewallPolicyRule, error) {
	klog.V(5).Infof("GCEAlphaNetworkFirewallPolicies.GetRule(%v, %v, %v, ...): called", ctx, key, priority)

	if !key.Valid() {
		klog.V(2).Infof("GCEAlphaNetworkFirewallPolicies.GetRule(%v, %v, %v, ...): key is invalid (%#v)", ctx, key, key, priority)
		return nil, fmt.Errorf("invalid GCE key (%+v)", key)
	}
	projectID := g.s.ProjectRouter.ProjectID(ctx, "alpha", "NetworkFirewallPolicies")
	rk := &RateLimitKey{
		ProjectID: projectID,
		Operation: "GetRule",
		Version:   meta.Version("alpha"),
		Service:   "NetworkFirewallPolicies",
	}
	klog.V(5).Infof("GCEAlphaNetworkFirewallPolicies.GetRule(%v, %v, %v, ...): projectID = %v, rk = %+v", ctx, key, priority, projectID, rk)

	if err := g.s.RateLimiter.Accept(ctx, rk); err != nil {
		klog.V(4).Infof("GCEAlphaNetworkFirewallPolicies.GetRule(%v, %v, %v, ...): RateLimiter error: %v", ctx, key, priority, err)
		return nil, err
	}
	call := g.s.Alpha.NetworkFirewallPolicies.GetRule(projectID, key.Name)
	if priority != nil {
		call = call.Priority(*priority)
	}
	call.Context(ctx)
	v, err := call.Do()
	klog.V(4).Infof("GCEAlphaNetworkFirewallPolicies.GetRule(%v, %v, %v, ...) = %+v, %v", ctx, key, priority, v, err)
	return v, err
}

// RemoveRule is a method on GCEAlphaNetworkFirewallPolicies.
func (g *GCEAlphaNetworkFirewallPolicies) RemoveRule(ctx context.Context, key *meta.Key, priority *int64) error {
	klog.V(5).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...): called", ctx, key, priority)

	if !key.Valid() {
		klog.V(2).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...): key is invalid (%#v)", ctx, key, priority, key)
		return fmt.Errorf("invalid GCE key (%+v)", key)
	}
	projectID := g.s.ProjectRouter.ProjectID(ctx, "alpha", "NetworkFirewallPolicies")
	rk := &RateLimitKey{
		ProjectID: projectID,
		Operation: "RemoveRule",
		Version:   meta.Version("alpha"),
		Service:   "NetworkFirewallPolicies",
	}
	klog.V(5).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...): projectID = %v, rk = %+v", ctx, key, priority, projectID, rk)

	if err := g.s.RateLimiter.Accept(ctx, rk); err != nil {
		klog.V(4).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...): RateLimiter error: %v", ctx, key, priority, err)
		return err
	}
	call := g.s.Alpha.NetworkFirewallPolicies.RemoveRule(projectID, key.Name)
	if priority != nil {
		call = call.Priority(*priority)
	}
	call.Context(ctx)
	op, err := call.Do()
	if err != nil {
		klog.V(4).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...) = %+v", ctx, key, priority, err)
		return err
	}
	err = g.s.WaitForCompletion(ctx, op)
	klog.V(4).Infof("GCEAlphaNetworkFirewallPolicies.RemoveRule(%v, %v, %v, ...) = %+v", ctx, key, priority, err)
	return err
}

type MockAlphaNetworkFirewallPoliciesOps struct {
	GetRuleHook            func(context.Context, *meta.Key, *int64, *MockAlphaNetworkFirewallPolicies) (*alpha.FirewallPolicyRule, error)
	RemoveRuleHook         func(context.Context, *meta.Key, *int64, *MockAlphaNetworkFirewallPolicies) error
}

// GetRule is a mock for the corresponding method.
func (m *MockAlphaNetworkFirewallPolicies) GetRule(ctx context.Context, key *meta.Key, priority *int64) (*alpha.FirewallPolicyRule, error) {
	if m.GetRuleHook != nil {
		return m.GetRuleHook(ctx, key, priority, m)
	}
	return nil, fmt.Errorf("GetRuleHook must be set")
}

// RemoveRule is a mock for the corresponding method.
func (m *MockAlphaNetworkFirewallPolicies) RemoveRule(ctx context.Context, key *meta.Key, priority *int64) error {
	if m.RemoveRuleHook != nil {
		return m.RemoveRuleHook(ctx, key, priority, m)
	}
	return nil
}
