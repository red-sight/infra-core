import { Injectable } from '@nestjs/common';
import { GateProvider } from './gate-providers/gate-provider';
import { KrakendGateProvider } from './gate-providers/krakend.gate-provider';

@Injectable()
export class GateService {
  provider: GateProvider;

  constructor() {
    this.provider = new KrakendGateProvider();
  }
}
