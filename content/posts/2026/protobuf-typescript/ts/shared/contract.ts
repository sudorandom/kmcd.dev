export interface BenchmarkAdapter {
  library: string;
  wireSize: number;
  encode(iterations: number): number;
  decode(iterations: number): number;
  verify(): boolean;
}
