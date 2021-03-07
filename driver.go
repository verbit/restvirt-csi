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
	"github.com/verbit/restvirt-client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Driver struct {
	c *restvirt.Client
	s *grpc.Server
}

type InstanceData struct {
	V1 struct {
		InstanceID string `json:"instance_id"`
	} `json:"v1"`
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
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

func (d *Driver) NodeGetInfo(ctx context.Context, request *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
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

func (d *Driver) NodeStageVolume(ctx context.Context, request *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
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

func (d *Driver) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
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

func (d *Driver) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
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

func (d *Driver) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
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

func (d *Driver) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func NewDriver(host string) (*Driver, error) {
	client, err := restvirt.NewClientFromEnvironment()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &Driver{
		c: client,
	}, nil
}

func (d *Driver) GetPluginInfo(ctx context.Context, request *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          "restvirt.verbit.io",
		VendorVersion: "0.0.1",
	}, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, request *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
					},
				},
			},
			// TODO: add online volume expansion once it's ready
		},
	}, nil
}

func (d *Driver) Probe(ctx context.Context, request *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{
		Ready: wrapperspb.Bool(true),
	}, nil
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, request *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	caps := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		// TODO
		// csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
		// csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}

	capabilities := make([]*csi.ControllerServiceCapability, len(caps))
	for i, c := range caps {
		capabilities[i] = &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: c,
				},
			},
		}
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: capabilities,
	}, nil
}

func (d *Driver) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// TODO: handle pre-populated?
	capacity := request.CapacityRange.GetRequiredBytes()
	vol := restvirt.Volume{
		Name: request.Name,
		Size: capacity,
	}
	id, err := d.c.CreateVolume(vol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: capacity,
			VolumeId:      id,
			// AccessibleTopology:   nil,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	err := d.c.DeleteVolume(request.VolumeId)
	return &csi.DeleteVolumeResponse{}, err
}

func (d *Driver) ControllerGetVolume(ctx context.Context, request *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	volume, err := d.c.GetVolume(request.VolumeId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.ControllerGetVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: volume.Size,
			VolumeId:      volume.ID,
		},
		Status: &csi.ControllerGetVolumeResponse_VolumeStatus{
			PublishedNodeIds: []string{}, // TODO: implement
		},
	}, nil
}

func (d *Driver) ListVolumes(ctx context.Context, request *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	// TODO: implement paging
	volumes, err := d.c.GetVolumes()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	volumeEntries := make([]*csi.ListVolumesResponse_Entry, len(volumes))
	for i, v := range volumes {
		volumeEntries[i] = &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				CapacityBytes: v.Size,
				VolumeId:      v.ID,
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				PublishedNodeIds: []string{}, // TODO: implement
			},
		}
	}

	return &csi.ListVolumesResponse{
		Entries:   volumeEntries,
		NextToken: "",
	}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	attachment, err := d.c.CreateAttachment(request.NodeId, request.VolumeId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			"disk_address": attachment.DiskAddress,
		},
	}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	err := d.c.DeleteAttachment(request.NodeId, request.VolumeId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, request *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) GetCapacity(ctx context.Context, request *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) CreateSnapshot(ctx context.Context, request *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, request *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(ctx context.Context, request *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, request *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	glog.V(3).Infof("GRPC call: %s", info.FullMethod)
	// TODO: glog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	glog.V(5).Infof("GRPC request: %s", req)
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC error: %v", err)
	} else {
		// TODO: glog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
		glog.V(5).Infof("GRPC response: %s", resp)
	}
	return resp, err
}
