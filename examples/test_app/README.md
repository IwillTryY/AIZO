# Test Application for RealityOS Containers

This is a simple test application that runs inside a container.

## Files

- `app.sh` - Main application script
- `README.md` - This file

## Usage

This directory is used by the container runtime to build an image.

```bash
runtime.BuildImage(ctx, "./test_app", "test-app", "v1.0")
```

The entire directory will be copied into the container's rootfs.
