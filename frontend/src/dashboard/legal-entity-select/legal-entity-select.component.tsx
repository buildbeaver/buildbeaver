import React, { useContext } from 'react';
import { Select, ISelectItem } from '../../components/select/select.component';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { LegalEntitiesContext } from '../../contexts/legal-entities/legal-entities.context';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';
import { IoAddCircleOutline } from 'react-icons/io5';

export function LegalEntitySelect(): JSX.Element {
  const { legalEntities } = useContext(LegalEntitiesContext);
  const { selectedLegalEntity, selectLegalEntity } = useContext(SelectedLegalEntityContext);

  const items: ISelectItem[] = [
    ...legalEntities.map((legalEntity: ILegalEntity) => {
      return {
        content: <span className="p-2">{legalEntity.name}</span>,
        label: legalEntity.name,
        onClick: () => selectLegalEntity(legalEntity)
      };
    }),
    {
      content: (
        <a
          className="flex grow gap-x-1 border-t p-2 text-blue-400"
          target="_blank"
          href="https://github.com/apps/buildbeaver"
          rel="noopener noreferrer"
        >
          <div>
            <IoAddCircleOutline size={20} />
          </div>
          <span>Add new</span>
        </a>
      ),
      label: 'Add new'
    }
  ];

  return (
    <div className="grow">
      <Select items={items} selectedItem={selectedLegalEntity.name} />
    </div>
  );
}
