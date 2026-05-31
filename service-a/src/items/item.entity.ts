import { ApiProperty } from '@nestjs/swagger';

export class Item {
  @ApiProperty({ example: 1 })
  id: number;

  @ApiProperty({ example: 'Widget' })
  name: string;

  @ApiProperty({ example: 'A useful widget', required: false })
  description?: string;
}
