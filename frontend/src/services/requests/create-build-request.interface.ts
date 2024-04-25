export interface ICreateBuildRequest {
  from_build_id: string;
  opts: IBuildOpts;
}

export interface IBuildOpts {
  force: boolean;
}
