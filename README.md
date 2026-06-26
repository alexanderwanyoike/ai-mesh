# ai-mesh

A multi-provider CLI for generating 3D meshes (GLB) from text or images with AI.

Supports [Fal](https://fal.ai/) (Tencent Hunyuan3D 3.1) and [Meshy](https://www.meshy.ai/) out of the box, with a clean provider interface for adding more. Built to be scripted, batched, and called by agents - one mesh per invocation, the output path on stdout, progress on stderr.

It is the mesh half of a composable pair with [ai-img](https://github.com/alexanderwanyoike/ai-img): `ai-img` turns a prompt into an image, `ai-mesh` turns an image into a mesh.

## Installation

### From release binaries

Download a prebuilt binary from the [Releases](https://github.com/alexanderwanyoike/ai-mesh/releases) page (Linux, macOS, Windows; amd64 and arm64).

### From source

```bash
go install github.com/alexanderwanyoike/ai-mesh@latest
```

### Build locally

```bash
git clone https://github.com/alexanderwanyoike/ai-mesh.git
cd ai-mesh
make build
```

## Quick Start

1. Set the API key for the provider you want:

```bash
export FAL_API_KEY="your-key-here"     # for fal
export MESHY_API_KEY="your-key-here"   # for meshy
```

2. Generate a mesh from an image:

```bash
ai-mesh -i character.png -o character.glb
```

3. Generate a mesh from a text prompt (Meshy):

```bash
ai-mesh -p meshy "a low-poly treasure chest" -o chest.glb
```

## Configuration

| Provider | Environment Variable | Get a key |
|----------|---------------------|-----------|
| Fal      | `FAL_API_KEY`       | [Fal dashboard](https://fal.ai/dashboard/keys) |
| Meshy    | `MESHY_API_KEY`     | [Meshy API](https://www.meshy.ai/api) |

## Usage

```
ai-mesh [flags] ["prompt"]
```

Provide a `"prompt"` (text-to-3d) or an `--input` image (image-to-3d). The result
is downloaded and written to the `--output` path; the path is echoed to stdout.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--provider` | `-p` | `fal` | Provider: `fal`, `meshy` |
| `--model` | `-m` | per-provider | Model name/ID or endpoint |
| `--output` | `-o` | `output.glb` | Output file path |
| `--input` | `-i` | | Input image for image-to-3d |
| `--faces` | | `50000` | Target face/polygon count |
| `--pbr` | | `false` | Request PBR material maps |

## Providers

### Fal (default) - image-to-3d

Tencent Hunyuan3D 3.1, hosted on Fal. Image input only; a bare text prompt returns
a hint to pass an image or use Meshy.

| Model (`-m`) | Description |
|--------------|-------------|
| `fal-ai/hunyuan-3d/v3.1/pro/image-to-3d` (default) | High quality, up to 1.5M faces |
| `fal-ai/hunyuan-3d/v3.1/rapid/image-to-3d` | Faster / cheaper |

### Meshy - text-to-3d and image-to-3d

Image input runs Meshy's single-stage image-to-3d. A text prompt runs the
two-stage text-to-3d (preview geometry, then refine with texture).

| Model (`-m`) | Description |
|--------------|-------------|
| `meshy-5` (default) | Meshy 5 |
| `meshy-6` | Meshy 6 |
| `latest` | Newest available |

## Composing

`ai-mesh` does one thing, so it pipes cleanly into batches, agents, and other tools.

### Text -> image -> mesh (with ai-img)

```bash
ai-img "a stone golem, T-pose, plain background" -o golem.png
ai-mesh -i golem.png -o golem.glb
```

### Batch a folder of images

```bash
for img in refs/*.png; do
  ai-mesh -i "$img" -o "meshes/$(basename "${img%.png}").glb"
done
```

### In a script / agent

stdout is just the output path, so it chains:

```bash
mesh=$(ai-mesh -i hero.png -o hero.glb)
blender --background --python import_glb.py -- "$mesh"
```

## Development

```bash
make build     # Build binary
make test      # Run tests
make clean     # Remove binary
make install   # go install
```

## License

MIT
