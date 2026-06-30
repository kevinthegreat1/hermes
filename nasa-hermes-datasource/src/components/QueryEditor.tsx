import React, { useCallback, useEffect, useState } from 'react';
import { Combobox, InlineField, MultiCombobox, RadioButtonGroup, Stack } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import type { ComboboxOption } from '@grafana/ui';
import { DataSource } from '../datasource';
import { MyDataSourceOptions, MyQuery, QueryType } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

const QUERY_TYPE_OPTIONS: Array<SelectableValue<QueryType>> = [
  { label: 'Telemetry', value: 'telemetry' },
  { label: 'Events', value: 'events' },
];

function toOptions(values: string[]): Array<ComboboxOption<string>> {
  return values.map((v) => ({ label: v, value: v }));
}

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const queryType = query.queryType ?? 'telemetry';

  // Telemetry state
  const [componentOptions, setComponentOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [channelOptions, setChannelOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [sourceOptions, setSourceOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [keyOptions, setKeyOptions] = useState<Array<ComboboxOption<string>>>([]);

  const [componentLoading, setComponentLoading] = useState(false);
  const [channelLoading, setChannelLoading] = useState(false);
  const [sourceLoading, setSourceLoading] = useState(false);
  const [keyLoading, setKeyLoading] = useState(false);

  // Event state
  const [eventComponentOptions, setEventComponentOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [eventNameOptions, setEventNameOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [eventSourceOptions, setEventSourceOptions] = useState<Array<ComboboxOption<string>>>([]);
  const [severityOptions, setSeverityOptions] = useState<Array<ComboboxOption<string>>>([]);

  const [eventComponentLoading, setEventComponentLoading] = useState(false);
  const [eventNameLoading, setEventNameLoading] = useState(false);
  const [eventSourceLoading, setEventSourceLoading] = useState(false);
  const [severityLoading, setSeverityLoading] = useState(false);

  // --- Telemetry data loading ---

  useEffect(() => {
    if (queryType !== 'telemetry') {
      return;
    }
    setComponentLoading(true);
    datasource
      .getComponents()
      .then((values) => setComponentOptions(toOptions(values)))
      .catch(() => setComponentOptions([]))
      .finally(() => setComponentLoading(false));
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'telemetry') {
      return;
    }
    setSourceLoading(true);
    datasource
      .getSources()
      .then((values) => setSourceOptions(toOptions(values)))
      .catch(() => setSourceOptions([]))
      .finally(() => setSourceLoading(false));
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'telemetry' || !query.component) {
      setChannelOptions([]);
      return;
    }
    setChannelLoading(true);
    datasource
      .getChannels(query.component)
      .then((values) => setChannelOptions(toOptions(values)))
      .catch(() => setChannelOptions([]))
      .finally(() => setChannelLoading(false));
  }, [datasource, queryType, query.component]);

  useEffect(() => {
    if (queryType !== 'telemetry' || !query.component || !query.channel) {
      setKeyOptions([]);
      return;
    }
    setKeyLoading(true);
    datasource
      .getKeys(query.component, query.channel)
      .then((values) => setKeyOptions(toOptions(values)))
      .catch(() => setKeyOptions([]))
      .finally(() => setKeyLoading(false));
  }, [datasource, queryType, query.component, query.channel]);

  // --- Event data loading ---

  useEffect(() => {
    if (queryType !== 'events') {
      return;
    }
    setEventComponentLoading(true);
    datasource
      .getEventComponents()
      .then((values) => setEventComponentOptions(toOptions(values)))
      .catch(() => setEventComponentOptions([]))
      .finally(() => setEventComponentLoading(false));
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'events') {
      return;
    }
    setEventSourceLoading(true);
    datasource
      .getEventSources()
      .then((values) => setEventSourceOptions(toOptions(values)))
      .catch(() => setEventSourceOptions([]))
      .finally(() => setEventSourceLoading(false));
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'events') {
      return;
    }
    setSeverityLoading(true);
    datasource
      .getEventSeverities()
      .then((options) => setSeverityOptions(options))
      .catch(() => setSeverityOptions([]))
      .finally(() => setSeverityLoading(false));
  }, [datasource, queryType]);

  useEffect(() => {
    if (queryType !== 'events' || !query.component) {
      setEventNameOptions([]);
      return;
    }
    setEventNameLoading(true);
    datasource
      .getEventNames(query.component)
      .then((values) => setEventNameOptions(toOptions(values)))
      .catch(() => setEventNameOptions([]))
      .finally(() => setEventNameLoading(false));
  }, [datasource, queryType, query.component]);

  // --- Handlers ---

  const onQueryTypeChange = useCallback(
    (value: QueryType) => {
      onChange({
        ...query,
        queryType: value,
        component: undefined,
        channel: undefined,
        key: undefined,
        source: undefined,
        eventName: undefined,
        severity: undefined,
      });
    },
    [onChange, query]
  );

  const onComponentChange = useCallback(
    (option: ComboboxOption<string>) => {
      if (queryType === 'events') {
        onChange({ ...query, component: option.value, eventName: undefined });
      } else {
        onChange({ ...query, component: option.value, channel: undefined, key: undefined });
      }
    },
    [onChange, query, queryType]
  );

  const onChannelChange = useCallback(
    (option: ComboboxOption<string>) => {
      const updated = { ...query, channel: option.value, key: undefined };
      onChange(updated);
      if (updated.component && updated.channel) {
        onRunQuery();
      }
    },
    [onChange, onRunQuery, query]
  );

  const onSourceChange = useCallback(
    (option: ComboboxOption<string> | null) => {
      const updated = { ...query, source: option?.value ?? undefined };
      onChange(updated);
      if (queryType === 'telemetry' && updated.component && updated.channel) {
        onRunQuery();
      }
      if (queryType === 'events' && updated.component && updated.eventName) {
        onRunQuery();
      }
    },
    [onChange, onRunQuery, query, queryType]
  );

  const onKeyChange = useCallback(
    (option: ComboboxOption<string> | null) => {
      const updated = { ...query, key: option?.value ?? undefined };
      onChange(updated);
      if (updated.component && updated.channel) {
        onRunQuery();
      }
    },
    [onChange, onRunQuery, query]
  );

  const onEventNameChange = useCallback(
    (option: ComboboxOption<string>) => {
      const updated = { ...query, eventName: option.value };
      onChange(updated);
      if (updated.component && updated.eventName) {
        onRunQuery();
      }
    },
    [onChange, onRunQuery, query]
  );

  const onSeverityChange = useCallback(
    (options: Array<ComboboxOption<string>>) => {
      const updated = { ...query, severity: options.length > 0 ? options.map((o) => o.value) : undefined };
      onChange(updated);
      if (updated.component && updated.eventName) {
        onRunQuery();
      }
    },
    [onChange, onRunQuery, query]
  );

  return (
    <Stack direction="column" gap={0}>
      <div style={{ marginBottom: 4 }}>
        <RadioButtonGroup
          id="query-editor-query-type"
          options={QUERY_TYPE_OPTIONS}
          value={queryType}
          onChange={onQueryTypeChange}
          size="sm"
        />
      </div>

      {queryType === 'telemetry' && (
        <>
          <Stack gap={0}>
            <InlineField label="Component" labelWidth={16} tooltip="FSW component or module" required>
              <Combobox
                id="query-editor-component"
                options={componentOptions}
                value={query.component ?? null}
                onChange={onComponentChange}
                loading={componentLoading}
                placeholder="Select component"
                width={28}
              />
            </InlineField>
            <InlineField label="Channel" labelWidth={16} tooltip="Telemetry channel name" required>
              <Combobox
                key={`channel-${query.component ?? ''}`}
                id="query-editor-channel"
                options={channelOptions}
                value={query.channel ?? null}
                onChange={onChannelChange}
                loading={channelLoading}
                disabled={!query.component}
                placeholder={query.component ? 'Select channel' : 'Select a component first'}
                width={28}
              />
            </InlineField>
          </Stack>
          <Stack gap={0}>
            <InlineField label="Source" labelWidth={16} tooltip="FSW source identifier (optional)">
              <Combobox
                id="query-editor-source"
                options={sourceOptions}
                value={query.source ?? null}
                onChange={onSourceChange}
                isClearable
                loading={sourceLoading}
                placeholder="All sources"
                width={28}
              />
            </InlineField>
            {keyOptions.length > 1 && (
              <InlineField label="Key" labelWidth={16} tooltip="Value field path for compound channels">
                <Combobox
                  id="query-editor-key"
                  options={keyOptions}
                  value={query.key ?? null}
                  onChange={onKeyChange}
                  isClearable
                  loading={keyLoading}
                  placeholder="All keys"
                  width={28}
                />
              </InlineField>
            )}
          </Stack>
        </>
      )}

      {queryType === 'events' && (
        <>
          <Stack gap={0}>
            <InlineField label="Component" labelWidth={16} tooltip="FSW component or module" required>
              <Combobox
                id="query-editor-event-component"
                options={eventComponentOptions}
                value={query.component ?? null}
                onChange={onComponentChange}
                loading={eventComponentLoading}
                placeholder="Select component"
                width={28}
              />
            </InlineField>
            <InlineField label="Event name" labelWidth={16} tooltip="Event name" required>
              <Combobox
                key={`event-name-${query.component ?? ''}`}
                id="query-editor-event-name"
                options={eventNameOptions}
                value={query.eventName ?? null}
                onChange={onEventNameChange}
                loading={eventNameLoading}
                disabled={!query.component}
                placeholder={query.component ? 'Select event' : 'Select a component first'}
                width={28}
              />
            </InlineField>
          </Stack>
          <Stack gap={0}>
            <InlineField label="Source" labelWidth={16} tooltip="FSW source identifier (optional)">
              <Combobox
                id="query-editor-event-source"
                options={eventSourceOptions}
                value={query.source ?? null}
                onChange={onSourceChange}
                isClearable
                loading={eventSourceLoading}
                placeholder="All sources"
                width={28}
              />
            </InlineField>
            <InlineField label="Severity" labelWidth={16} tooltip="Filter by event severity (optional)">
              <MultiCombobox
                id="query-editor-severity"
                options={severityOptions}
                value={query.severity ?? []}
                onChange={onSeverityChange}
                loading={severityLoading}
                isClearable
                placeholder="All severities"
                width={28}
              />
            </InlineField>
          </Stack>
        </>
      )}
    </Stack>
  );
}
