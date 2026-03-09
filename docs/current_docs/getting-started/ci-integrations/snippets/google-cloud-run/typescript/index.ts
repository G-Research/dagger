import { dag, Secret, object, func } from "@github.com/G-Research/dagger"

@object()
class MyModule {
  @func()
  async deploy(
    projectName: string,
    serviceLocation: string,
    imageAddress: string,
    servicePort: number,
    credential: Secret,
  ): Promise<string> {
    return await dag
      .googleCloudRun()
      .createService(
        projectName,
        serviceLocation,
        imageAddress,
        servicePort,
        credential,
      )
  }
}
