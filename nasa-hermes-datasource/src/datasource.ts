import { DataQueryRequest, DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { map } from 'rxjs/operators';
import { MyQuery, MyDataSourceOptions, DEFAULT_QUERY, ChannelRef } from './types';

export class DataSource extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
  }

  query(request: DataQueryRequest<MyQuery>) {
    return super.query(request).pipe(
      map((response) => {
        for (const result of response.data) {
          const query = request.targets.find((t) => t.refId === result.refId);
          if (query?.queryType === 'events' && query.sources.length) {
            result.fields = result.fields.filter((f: { name: string }) => f.name !== 'source');
          }
        }
        return response;
      })
    );
  }

  getDefaultQuery(_: CoreApp): Partial<MyQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: MyQuery, scopedVars: ScopedVars) {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      queryType: query.queryType ?? 'telemetry',
      channels: query.channels.map(ch => ({
        component: templateSrv.replace(ch.component, scopedVars),
        name: templateSrv.replace(ch.name, scopedVars),
      })),
      sources: query.sources.map(s => templateSrv.replace(s, scopedVars)),
      keys: query.keys.map(k => templateSrv.replace(k, scopedVars)),
      timeOverrideFrom: query.timeOverrideFrom,
      timeOverrideTo: query.timeOverrideTo,
      timeField: query.timeField ?? 'time'
    };
  }

  filterQuery(query: MyQuery): boolean {
    if (query.queryType === 'events') {
      return true;
    }
    return !!query.channels.length;
  }

  // Telemetry resources
  async getChannels(): Promise<ChannelRef[]> {
    return this.getResource('telemetry/channels');
  }

  async getSources(): Promise<string[]> {
    return this.getResource('telemetry/sources');
  }

  async getKeys(channels: ChannelRef[]): Promise<string[]> {
    const components = [...new Set(channels.map(ch => ch.component))];
    const names = channels.map(ch => ch.name);
    return this.getResource('telemetry/keys', { components, channels: names });
  }

  // Event resources
  async getEventSources(): Promise<string[]> {
    return this.getResource('events/sources');
  }
}
