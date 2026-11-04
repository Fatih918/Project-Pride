# PRIDE Plugin Module

**Status:** Foundation design proposed; implementation not yet started.

The Plugin Module is the platform domain responsible for installed Plugin records, registry state, loading, lifecycle, and Active Plugin runtime execution.

The first implementation increment is limited to read-only discovery. It will implement the existing `sdks/plugin/server.DiscoveryPort` contract without adding lifecycle or execution RPCs.

## Architecture

See [Plugin Module Foundation Design](../../01-Architecture/Plugin%20Module%20Foundation%20Design.md) for:
- ownership and dependency boundaries;
- local installed Plugin Registry vs official published Registry Repository;
- domain data boundaries;
- five discovery operations;
- proposed package layout;
- staged implementation plan and acceptance criteria.

## Current Scope

No production persistence adapter, package loader, lifecycle manager, or Plugin runtime is provided by this initial foundation. Those components will be introduced only after their contracts and security requirements are defined.

Core owns capability matching and Plugin selection. The Plugin Module exposes installed Plugin declarations and operational state through the Plugin SDK. Permission IDs are metadata; authorization remains outside the Module's discovery layer.
