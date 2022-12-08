import { css } from '@emotion/css';
import React from 'react';
import { useParams } from 'react-router-dom';

import { LoadingPlaceholder, useStyles2 } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

import { useGetSingle } from './api';
import { DetailsOverview, DetailsStatus, DetailsHeaderActions } from './components';
import { tabIds, usePluginRecipeDetailsPageTabs } from './hooks';

const navId = 'connections-plugin-recipes';

export function PluginRecipeDetailsPage() {
  const params = useParams<{ id: string }>();
  const { status, error, data } = useGetSingle(params.id);
  const { tabId, tabs } = usePluginRecipeDetailsPageTabs();
  const styles = useStyles2(getStyles);
  const onStartInstall = () => {}; // called when the user clicks on "Install"
  const info = [
    { label: 'Version', value: 'v1.0.0' },
    { label: 'Rating', value: '4/5' },
  ];

  if (status === 'loading') {
    return (
      <Page navId={navId} pageNav={{ text: '', subTitle: '', active: true }}>
        <Page.Contents>
          <LoadingPlaceholder text="Loading..." />
        </Page.Contents>
      </Page>
    );
  }

  if (status === 'error') {
    return (
      <Page navId={navId} pageNav={{ text: 'Error', subTitle: '', active: true }}>
        <Page.Contents>
          <p>{String(error)}</p>
        </Page.Contents>
      </Page>
    );
  }

  if (status === 'success' && !data) {
    return (
      <Page navId={navId} pageNav={{ text: '', subTitle: '', active: true }}>
        <Page.Contents>Plugin recipe not found.</Page.Contents>
      </Page>
    );
  }

  return (
    <Page
      navId={navId}
      pageNav={{ text: data.name, subTitle: data.summary, active: true, children: tabs }}
      actions={<DetailsHeaderActions onInstall={onStartInstall} />}
      info={info}
      renderTitle={(title) => (
        <div className={styles.pageTitleContainer}>
          <img className={styles.pageTitleImage} src={data.logo} alt={`Logo of ${data.name}`} />
          <h1 className={styles.pageTitle}>{title}</h1>
        </div>
      )}
    >
      <Page.Contents>
        <div className={styles.content}>
          {tabId === tabIds.overview && <DetailsOverview recipe={data} />}
          {tabId === tabIds.status && <DetailsStatus recipe={data} />}
        </div>
      </Page.Contents>
    </Page>
  );
}

const getStyles = () => ({
  content: css`
    min-width: 900px;
    width: 60%;
  `,
  pageTitleContainer: css`
    display: flex;
    align-items: center;
  `,
  pageTitleImage: css`
    width: 40px;
    height: 40px;
    margin-left: 5px;
    margin-right: 10px;
  `,
  pageTitle: css`
    margin-bottom: 0;
  `,
});