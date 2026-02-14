import { Module, OnModuleInit } from '@nestjs/common';
import { MainService } from './main.service';
import { DockerModule } from '../docker/docker.module';
import { OpenapiModule } from '../openapi/openapi.module';
import { GateModule } from '../gate/gate.module';
import { DockerService } from '../docker/docker.service';

@Module({
  providers: [MainService],
  imports: [DockerModule, OpenapiModule, GateModule],
})
export class MainModule implements OnModuleInit {
  constructor(
    private readonly mainService: MainService,
    private readonly dockerService: DockerService,
  ) {}

  async onModuleInit() {
    await this.mainService.bootstrap();
    await this.dockerService.watch();
  }
}
