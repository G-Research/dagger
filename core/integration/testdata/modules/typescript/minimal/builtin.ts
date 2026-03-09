import { Directory, object, func } from "@github.com/G-Research/dagger"

@object()
export class Minimal {
  @func()
  async read(dir: Directory): Promise<string> {
    return dir.file("foo").contents()
  }


  @func()
  async readSlice(dir: Directory[]): Promise<string> {
    return dir[0].file("foo").contents()
  }

  @func()
  async readVariadic(...dir: Directory[]): Promise<string> {
    return dir[0].file("foo").contents()
  }

  @func()
  async readOptional(dir?: Directory): Promise<string> {
    if (!dir) {
      return ""
    }

    return dir.file("foo").contents()
  }
}
