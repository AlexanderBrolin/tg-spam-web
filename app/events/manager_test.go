package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelManagerNew(t *testing.T) {
	mgr := NewChannelManager()
	assert.NotNil(t, mgr)
	assert.Empty(t, mgr.Running())
}

func TestChannelManagerAddDuplicate(t *testing.T) {
	mgr := NewChannelManager()

	// we can't fully test AddChannel without a real TbAPI, but we can test duplicate detection
	// by adding a mock entry directly
	mgr.channels["test-gid"] = &runningChannel{
		config: ChannelConfig{GID: "test-gid"},
		done:   make(chan struct{}),
	}

	err := mgr.AddChannel(ChannelConfig{GID: "test-gid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestChannelManagerRemoveNonExistent(t *testing.T) {
	mgr := NewChannelManager()

	err := mgr.RemoveChannel("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestChannelManagerRunning(t *testing.T) {
	mgr := NewChannelManager()

	done := make(chan struct{})
	mgr.channels["gid-1"] = &runningChannel{
		config: ChannelConfig{GID: "gid-1"},
		done:   done,
	}
	mgr.channels["gid-2"] = &runningChannel{
		config: ChannelConfig{GID: "gid-2"},
		done:   done,
	}

	running := mgr.Running()
	assert.Len(t, running, 2)
	assert.Contains(t, running, "gid-1")
	assert.Contains(t, running, "gid-2")
}
