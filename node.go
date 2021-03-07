package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeServer struct{}

func (s *NodeServer) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}

func (s *NodeServer) NodeGetInfo(ctx context.Context, request *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	instanceDataRaw, err := ioutil.ReadFile("/var/run/cloud-init/instance-data.json")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "couldn't read instance data: %v", err)
	}

	var instanceData InstanceData
	err = json.Unmarshal(instanceDataRaw, &instanceData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "couldn't decode instance data: %v", err)
	}

	return &csi.NodeGetInfoResponse{
		NodeId:            instanceData.V1.InstanceID,
		MaxVolumesPerNode: 0,
		AccessibleTopology: &csi.Topology{Segments: map[string]string{
			"host": "patch.place", // TODO: obtain this from cloud_init or API call
		}},
	}, nil
}

func (s *NodeServer) NodeStageVolume(ctx context.Context, request *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if request.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "no volume id set")
	}

	if request.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "no staging target path set")
	}

	if request.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "no volume capability")
	}

	disk_address := request.PublishContext["disk_address"]
	disk_path := fmt.Sprintf("/dev/disk/by-path/%s", disk_address)

	mount := request.VolumeCapability.GetMount()
	if mount == nil {
		return &csi.NodeStageVolumeResponse{}, nil
	}

	fsType := mount.FsType
	if fsType == "" {
		fsType = "ext4"
	}

	cmd := exec.Command("mkfs."+fsType, disk_path)
	out, err := cmd.CombinedOutput()
	glog.V(3).Infof("mkfs output: %s", out)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	err = os.MkdirAll(request.StagingTargetPath, os.ModeDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	cmd = exec.Command("mount", disk_path, request.StagingTargetPath)
	out, err = cmd.CombinedOutput()
	glog.V(3).Infof("mount output: %s", out)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (s *NodeServer) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if request.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "no volume id set")
	}

	if request.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "no staging target path set")
	}

	cmd := exec.Command("umount", request.StagingTargetPath)
	out, err := cmd.CombinedOutput()
	glog.V(3).Infof("umount output: %s", out)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (s *NodeServer) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if request.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "no volume id set")
	}

	if request.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "no staging target path set")
	}

	if request.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "no target path set")
	}

	if request.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "no volume capability")
	}

	err := os.MkdirAll(request.TargetPath, os.ModeDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	cmd := exec.Command("mount", "-o", "bind", request.StagingTargetPath, request.TargetPath)
	out, err := cmd.CombinedOutput()
	glog.V(3).Infof("mount output: %s", out)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mount bind error: %v", err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (s *NodeServer) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if request.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "no volume id set")
	}

	if request.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "no target path set")
	}

	cmd := exec.Command("umount", request.TargetPath)
	out, err := cmd.CombinedOutput()
	glog.V(3).Infof("umount output: %s", out)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (s *NodeServer) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *NodeServer) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
