# Contributing to claude-coord

Thanks for your interest in improving claude-coord!

## Development Setup

```bash
# Clone the repo
git clone https://github.com/LoomLabs-Venture-Studio/claude-coord
cd claude-coord

# Build
make build

# Run tests
make test

# Run demo
make demo
```

## Project Structure

```
claude-coord/
├── cmd/claude-coord/     # CLI entry point
├── internal/
│   ├── cli/              # Command implementations
│   ├── config/           # Configuration handling
│   ├── lock/             # Lock management
│   └── agent/            # Agent lifecycle
├── examples/             # Example configurations
└── scripts/              # Build/install scripts
```

## Making Changes

1. Fork the repo
2. Create a feature branch: `git checkout -b feature/my-change`
3. Make your changes
4. Add tests if applicable
5. Run `make test` and `make lint`
6. Commit with a clear message
7. Push and open a PR

## Code Style

- Follow standard Go conventions
- Use meaningful variable names
- Add comments for non-obvious logic
- Keep functions focused and small

## Testing

- Add tests for new functionality
- Ensure existing tests pass
- Test edge cases (stale locks, race conditions, etc.)

## Areas for Contribution

- **Windows support**: `O_EXCL` equivalent for Windows
- **Better glob patterns**: More sophisticated pattern matching
- **Notifications**: Alert waiting agents when locks release
- **Metrics/logging**: Better observability
- **IDE integrations**: VS Code extension, etc.

## Questions?

Open an issue or discussion!
