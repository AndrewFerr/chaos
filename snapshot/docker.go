package snapshot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/element-hq/chaos/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const SnapshotTypeDocker = "docker"

type DockerConfig struct {
	ContainerName string `yaml:"container_name"`
}

type DockerSnapshotter struct {
	apiClient     *client.Client
	hsConfig      config.HomeserverConfig
	containerName string
}

func NewDockerSnapshotter(hsc config.HomeserverConfig) (Snapshotter, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client from env: %s", err)
	}
	snapshotConfig, err := config.UnmarshalInto[DockerConfig](hsc.Snapshot.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}
	return &DockerSnapshotter{
		apiClient:     apiClient,
		containerName: snapshotConfig.ContainerName,
		hsConfig:      hsc,
	}, nil
}

func (s *DockerSnapshotter) Snapshot() (*Snapshot, error) {
	reader, err := s.apiClient.ContainerStats(context.Background(), s.containerName, client.ContainerStatsOptions{})
	if err != nil {
		return nil, fmt.Errorf("ContainerStatsOneShot: %s", err)
	}
	defer reader.Body.Close()
	data := container.StatsResponse{}
	err = json.NewDecoder(reader.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("erorr decoding stats response: %s", err)
	}
	return &Snapshot{
		ProcessEntries: []ProcessSnapshot{
			{
				Homeserver:  s.hsConfig.Domain,
				ProcessName: s.containerName,
				MemoryBytes: int64(data.MemoryStats.Usage),
				MilliCPUs:   int64((data.CPUStats.CPUUsage.TotalUsage / 1000) / 1000), // ns -> ms
			},
		},
	}, nil
}
