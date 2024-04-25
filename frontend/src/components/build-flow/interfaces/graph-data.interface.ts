import { JobNode } from '../models/job-node.model';
import { ElkExtendedEdge, ElkNode } from 'elkjs/lib/elk.bundled';

export interface IGraphData {
  elkEdges: ElkExtendedEdge[];
  elkNodes: ElkNode[];
  jobNodes: JobNode[];
}
