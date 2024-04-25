<p align="center">
  <img width="240" height="240" src="https://github.com/buildbeaver/go-sdk/assets/1029312/e2276e36-8b3f-4e7d-b9b4-a1e384b28ac1">
</p>

---

# BuildBeaver Full Source Code

This repo contains the BuildBeaver free and open-source Continuous Integration system.
This includes the build server, build runner, Web UI, and the bb command line tool.

Tutorials and Guides can be found on the [BuildBeaver Documentation](https://buildbeaver.github.io/) site.

See [Releases](https://github.com/buildbeaver/bb-cli/releases) for command-line tool **pre-built binaries**
for Linux, Mac and Windows.

See [Building the BuildBeaver Server, CLI and Web UI](building-bb.md) for instructions on building from source.

## What is BuildBeaver?

The *BuildBeaver Team* are a group of DevOps and Build System enthusiasts who believe there is a better way to
define builds and CI/CD pipelines. Better than a complex spaghetti of YAML and scripts that only the Build Expert
on the team dares to touch.

Our aim is to help you *ship better software, faster.*

We've created a build tool that reflects the principles we see as the future of build and CI/CD systems:

- **Use a Real Language**: Use the same language for your software and your build, so any team member can easily
  work on the build. The *BuildBeaver SDK* makes it simple to define your build jobs, but with all the power of a
  procedural language.

- **Fully Dynamic Builds**: Add new jobs to a build at any time; build code can check the results of previous jobs,
  make API calls, or anything else - and then submit the next jobs to be run within the build. Rather than generating
  a build pipeline or workflow that then becomes read-only, new jobs can be added to existing workflows at any time.

- **Run anywhere**: Run the same build on your laptop or your CI system. Test builds locally before pushing changes
  to the build code.

- **Native or Docker Jobs**: Run build jobs using native runners on Mac, Windows or Linux, as well as
  Docker containers.

- **Fingerprinting**: Speed up builds by re-using artifacts from previous builds when the inputs to a Job haven't
  changed.

See the [Dynamic Builds Guide](https://buildbeaver.github.io/docs/category/guide-to-dynamic-builds) for details.
