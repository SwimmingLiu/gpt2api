<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import * as statsApi from '@/api/stats'
import * as poolApi from '@/api/accountPools'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const canWrite = computed(() => userStore.hasPerm('model:write'))

const models = ref<statsApi.Model[]>([])
const pools = ref<poolApi.AccountPool[]>([])
const routes = ref<poolApi.ModelPoolRoute[]>([])
const saving = ref<number | null>(null)

const formMap = reactive<Record<number, { pool_id: number; fallback_pool_id: number; enabled: boolean }>>({})

async function load() {
  const [m, p, r] = await Promise.all([
    statsApi.listModels(),
    poolApi.listAccountPools(),
    poolApi.listModelPoolRoutes(),
  ])
  models.value = m.items || []
  pools.value = p.items || []
  routes.value = r.items || []

  for (const model of models.value) {
    const route = routes.value.find((x) => x.model_id === model.id)
    formMap[model.id] = {
      pool_id: route?.pool_id || 0,
      fallback_pool_id: route?.fallback_pool_id || 0,
      enabled: route?.enabled ?? true,
    }
  }
}

async function saveRoute(modelID: number) {
  const form = formMap[modelID]
  if (!form?.pool_id) {
    ElMessage.warning('请先选择主账号池')
    return
  }
  saving.value = modelID
  try {
    await poolApi.putModelPoolRoute(modelID, form)
    ElMessage.success('模型池路由已保存')
  } finally {
    saving.value = null
  }
}

async function clearRoute(modelID: number) {
  await poolApi.deleteModelPoolRoute(modelID)
  formMap[modelID] = { pool_id: 0, fallback_pool_id: 0, enabled: true }
  ElMessage.success('模型池路由已删除')
}

onMounted(load)
</script>

<template>
  <div class="page-container">
    <div class="card-block">
      <h2 class="page-title">模型池路由</h2>
      <div class="page-sub">把模型固定路由到指定账号池。未配置路由时，后端继续保持旧的全局兼容行为。</div>
    </div>

    <div class="card-block">
      <el-table :data="models" row-key="id" table-layout="auto">
        <el-table-column prop="slug" label="模型" min-width="180" />
        <el-table-column prop="type" label="类型" width="100" />
        <el-table-column label="主池" min-width="180">
          <template #default="{ row }">
            <el-select v-model="formMap[row.id].pool_id" :disabled="!canWrite" clearable placeholder="未配置">
              <el-option v-for="p in pools" :key="p.id" :label="`${p.name} (${p.code})`" :value="p.id" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="兜底池" min-width="180">
          <template #default="{ row }">
            <el-select v-model="formMap[row.id].fallback_pool_id" :disabled="!canWrite" clearable placeholder="无">
              <el-option v-for="p in pools" :key="p.id" :label="`${p.name} (${p.code})`" :value="p.id" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">
            <el-switch v-model="formMap[row.id].enabled" :disabled="!canWrite" />
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :loading="saving === row.id" @click="saveRoute(row.id)">保存</el-button>
            <el-button link type="danger" @click="clearRoute(row.id)">清除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>
