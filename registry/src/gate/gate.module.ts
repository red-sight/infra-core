import { Module } from '@nestjs/common';
import { GateService } from './gate.service';

@Module({
  providers: [GateService],
  exports: [GateService],
})
export class GateModule {}
