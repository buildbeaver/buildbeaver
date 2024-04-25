import React from 'react';
import { Menu } from '../../components/menu/menu.component';
import { IoMenuSharp } from 'react-icons/io5';
import { IJobGraph } from '../../interfaces/job-graph.interface';
import { useLogDescriptor } from '../../hooks/log-descriptor/log-descriptor.hook';

interface Props {
  jobGraph: IJobGraph;
}

export function LogViewerMenu(props: Props): JSX.Element {
  const { jobGraph } = props;
  const { logDescriptor } = useLogDescriptor(jobGraph.job.log_descriptor_url);

  if (!logDescriptor) {
    return <></>;
  }

  const menuItems = [
    {
      content: (
        <a download href={`${logDescriptor.data_url}?expand=true&plaintext=true&download=true`}>
          Download logs
        </a>
      ),
      key: 'download-logs'
    }
  ];

  return (
    <Menu items={menuItems}>
      <IoMenuSharp size={22} />
    </Menu>
  );
}
