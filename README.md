# stodo

simple cli todo app that store the list in the current directory (useful for isolating todo per project)

## Install

```bash
go install github.com/luberius/stodo@latest
```

Or from source:
```bash
git clone https://github.com/luberius/stodo.git
cd stodo
go build
```

## Usage

```bash
stodo
```

### Keys

- `i`: Insert mode
- `ESC`: Normal mode
- `j/k`: Down/up
- `Space`: Toggle done
- `d`: Delete
- `w`: Save
- `q`: Quit

## License

MIT
