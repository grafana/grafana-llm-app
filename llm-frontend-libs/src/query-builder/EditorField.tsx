import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Field, Icon, PopoverContent, stylesFactory, Tooltip, useTheme2, ReactUtils } from '@grafana/ui';
import React, { ComponentProps } from 'react';

import { Space } from './Space';

interface EditorFieldProps extends ComponentProps<typeof Field>{
  label: string;
  children: React.ReactElement;
  width?: number | string;
  optional?: boolean;
  tooltip?: PopoverContent;
}

export const EditorField: React.FC<EditorFieldProps> = (props) => {
  const { label, optional, tooltip, children, width, ...fieldProps } = props;

  const theme = useTheme2();
  const styles = getStyles(theme, width);
  const childInputId = ReactUtils.getChildId(children);

  const labelEl = (
    <>
      <label className={styles.label} htmlFor={childInputId}>
        {label}
        {optional && <span className={styles.optional}> - optional</span>}
        {tooltip && (
          <Tooltip placement="top" content={tooltip} theme="info">
            <Icon name="info-circle" size="sm" className={styles.icon} />
          </Tooltip>
        )}
      </label>
      <Space v={0.5} />
    </>
  );

  const StyledChildren = () => {
    return <div className={styles.child}>{children}</div>
  } 

  return (
    <div className={styles.root}>
      <Field className={styles.field} label={labelEl} {...fieldProps}>
        <StyledChildren />
      </Field>
    </div>
  );
};

const getStyles = stylesFactory((theme: GrafanaTheme2, width?: number | string) => {
  return {
    root: css({
      minWidth: theme.spacing(width ?? 0),
    }),
    label: css({
      fontSize: 12,
      fontWeight: theme.typography.fontWeightMedium,
    }),
    optional: css({
      fontStyle: 'italic',
      color: theme.colors.text.secondary,
    }),
    field: css({
      marginBottom: 0, // GrafanaUI/Field has a bottom margin which we must remove
    }),

    // TODO: really poor hack to align the switch
    // Find a better solution to this
    child: css({
      display: 'flex',
      alignItems: 'center',
      minHeight: 30,
    }),
    icon: css({
      color: theme.colors.text.secondary,
      marginLeft: theme.spacing(1),
      ':hover': {
        color: theme.colors.text.primary,
      },
    }),
  };
});
