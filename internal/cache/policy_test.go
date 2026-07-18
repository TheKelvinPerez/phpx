package cache_test

import (
	"errors"
	"testing"

	"github.com/elefantephp/elefante/internal/cache"
	"github.com/elefantephp/elefante/internal/model"
)

func TestOfflinePolicyUsesDeterministicCachedSubstituteOrFails(t *testing.T) {
	t.Parallel()

	policy := cache.NetworkPolicy{
		Command: cache.CommandPlan,
		Offline: true,
	}
	artifact := cache.Artifact{
		Identity: "sha256:metadata",
		Path:     "/cache/metadata/sha256_metadata/response.json",
		SHA256:   "sha256:content",
	}
	decision, err := policy.Resolve(cache.NetworkRequest{
		Requirement: model.NetworkRead,
		Substitute:  &artifact,
	})
	if err != nil {
		t.Fatalf("resolve cached substitute: %v", err)
	}
	if decision.UseNetwork || decision.Artifact == nil ||
		*decision.Artifact != artifact {
		t.Fatalf("expected exact cached substitute, got %#v", decision)
	}

	_, err = policy.Resolve(cache.NetworkRequest{
		Requirement: model.NetworkRead,
	})
	var commandError *model.Error
	if !errors.As(err, &commandError) ||
		commandError.Code != model.ErrorNetwork {
		t.Fatalf("expected offline network error, got %v", err)
	}
	if model.ExitCode(err) != 8 {
		t.Fatalf("expected exit 8, got %d", model.ExitCode(err))
	}
}

func TestNetworkPolicyIsCommandSpecific(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		policy      cache.NetworkPolicy
		request     cache.NetworkRequest
		wantNetwork bool
		wantCode    model.ErrorCode
	}{
		{
			name:   "doctor never initiates network",
			policy: cache.NetworkPolicy{Command: cache.CommandDoctor},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRead,
			},
			wantCode: model.ErrorNetwork,
		},
		{
			name:   "plan reads metadata",
			policy: cache.NetworkPolicy{Command: cache.CommandPlan},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRead,
			},
			wantNetwork: true,
		},
		{
			name:   "plan does not download required artifacts",
			policy: cache.NetworkPolicy{Command: cache.CommandPlan},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRequired,
			},
			wantCode: model.ErrorNetwork,
		},
		{
			name:   "sync requires an approved source",
			policy: cache.NetworkPolicy{Command: cache.CommandSync},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRequired,
			},
			wantCode: model.ErrorNetwork,
		},
		{
			name:   "sync uses an approved source",
			policy: cache.NetworkPolicy{Command: cache.CommandSync},
			request: cache.NetworkRequest{
				Requirement:    model.NetworkRequired,
				ApprovedSource: true,
			},
			wantNetwork: true,
		},
		{
			name:   "run leaves network to its child",
			policy: cache.NetworkPolicy{Command: cache.CommandRun},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRead,
			},
			wantCode: model.ErrorNetwork,
		},
		{
			name:   "tool uses network on cache miss",
			policy: cache.NetworkPolicy{Command: cache.CommandTool},
			request: cache.NetworkRequest{
				Requirement: model.NetworkRequired,
			},
			wantNetwork: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			decision, err := test.policy.Resolve(test.request)
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("resolve policy: %v", err)
				}
				if decision.UseNetwork != test.wantNetwork {
					t.Fatalf("unexpected decision %#v", decision)
				}

				return
			}

			var commandError *model.Error
			if !errors.As(err, &commandError) ||
				commandError.Code != test.wantCode {
				t.Fatalf("expected %s, got %v", test.wantCode, err)
			}
		})
	}
}
