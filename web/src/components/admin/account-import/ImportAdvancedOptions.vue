<script setup lang="ts">
import type { ImportAdvancedOptions, SelectOption } from './types'

const model = defineModel<ImportAdvancedOptions>({
  required: true,
})

withDefaults(
  defineProps<{
    proxyOptions?: SelectOption[]
    poolOptions?: SelectOption[]
    disabled?: boolean
  }>(),
  {
    proxyOptions: () => [],
    poolOptions: () => [],
    disabled: false,
  },
)
</script>

<template>
  <el-form label-width="120px">
    <el-form-item label="更新已有邮箱">
      <el-switch v-model="model.update_existing" :disabled="disabled" />
    </el-form-item>

    <el-form-item label="默认代理">
      <el-select
        v-model="model.default_proxy_id"
        clearable
        filterable
        placeholder="不指定"
        :disabled="disabled"
        style="width: 100%"
      >
        <el-option
          v-for="item in proxyOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
          :disabled="item.disabled"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="导入到账号池">
      <el-select
        v-model="model.target_pool_id"
        clearable
        filterable
        placeholder="不加入账号池"
        :disabled="disabled"
        style="width: 100%"
      >
        <el-option
          v-for="item in poolOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
          :disabled="item.disabled"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="尝试补全身份">
      <el-switch v-model="model.resolve_identity" :disabled="disabled" />
    </el-form-item>

    <el-form-item label="导入后刷新">
      <el-switch v-model="model.kick_refresh" :disabled="disabled" />
    </el-form-item>

    <el-form-item label="导入后探测额度">
      <el-switch v-model="model.kick_quota_probe" :disabled="disabled" />
    </el-form-item>
  </el-form>
</template>
