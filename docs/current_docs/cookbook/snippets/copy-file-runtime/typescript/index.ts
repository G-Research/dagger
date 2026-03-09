import { object, func, File } from "@github.com/G-Research/dagger"
import * as fs from "fs"

@object()
class MyModule {
  // Copy a file to the Dagger module runtime container for custom processing
  @func()
  async copyFile(source: File) {
    await source.export("foo.txt")
    // your custom logic here
    // for example, read and print the file in the Dagger Engine container
    console.log(fs.readFileSync("foo.txt", "utf8"))
  }
}
