package main

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/verbit/restvirt-client"
	"github.com/verbit/restvirt-client/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func NewDriver() (*Driver, error) {
	client, err := restvirt.NewClientFromEnvironment()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	return &Driver{
		c: client,
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
	response, err := d.c.VolumeServiceClient.ListVolumes(ctx, &pb.ListVolumesRequest{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error: %v", err)
	}

	volumes := response.Volumes

	volumeEntries := make([]*csi.ListVolumesResponse_Entry, len(volumes))
	for i, v := range volumes {
		volumeEntries[i] = &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				CapacityBytes: int64(v.Size),
				VolumeId:      v.Id,
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
