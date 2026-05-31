import { ApiProperty } from '@nestjs/swagger';
import { IsString, IsNotEmpty, IsOptional } from 'class-validator';

export class CreateItemDto {
  @ApiProperty({ example: 'Widget' })
  @IsString()
  @IsNotEmpty()
  name: string;

  @ApiProperty({ example: 'A useful widget', required: false })
  @IsString()
  @IsOptional()
  description?: string;
}
