import { Module } from '@nestjs/common';
import { OpenapiService } from './openapi.service';

@Module({
  providers: [OpenapiService],
  exports: [OpenapiService],
})
export class OpenapiModule {}
